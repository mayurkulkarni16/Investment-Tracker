package services

import (
	"context"
	"crypto/sha1"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"investment-tracker/models"
	"investment-tracker/repository"
)

const maxImportRows = 5000

var headerCleaner = regexp.MustCompile(`[^a-z0-9]`)

// Column aliases (normalised: lowercase alphanumerics) for Zerodha Console and Groww exports.
var importColumns = map[string][]string{
	"symbol":   {"symbol", "instrument", "tradingsymbol", "scripcode"},
	"name":     {"stockname", "name", "companyname", "scripname"},
	"isin":     {"isin"},
	"qty":      {"qty", "quantity", "quantityavailable", "qtyavailable", "totalquantity"},
	"avg":      {"avgcost", "averageprice", "averagebuyprice", "avgprice", "avgbuyprice", "buyavg"},
	"type":     {"tradetype", "type", "transactiontype", "buysell", "side"},
	"price":    {"price", "tradeprice", "rate"},
	"value":    {"value", "amount", "tradevalue"},
	"date":     {"tradedate", "executiondateandtime", "orderexecutiontime", "date", "executiontime"},
	"tradeid":  {"tradeid", "exchangeorderid", "orderid", "exchangetradeid"},
	"status":   {"orderstatus", "status"},
	"exchange": {"exchange"},
	"segment":  {"segment"},
}

var importDateLayouts = []string{
	"2006-01-02", "02-01-2006", "02/01/2006", "2006/01/02", "2006-01-02T15:04:05", "2006-01-02 15:04:05",
	"02-01-2006 15:04", "02-01-2006 03:04 PM", "02 Jan 2006, 03:04 PM", "02 Jan 2006 03:04 PM", "02 Jan 2006", "02-Jan-2006",
}

type BrokerImportService struct {
	stocks *StockService
	repo   *repository.StockRepo
}

func NewBrokerImportService(stocks *StockService, repo *repository.StockRepo) *BrokerImportService {
	return &BrokerImportService{stocks: stocks, repo: repo}
}

type importTable struct {
	cols map[string]int
	rows [][]string
}

func (t importTable) get(row []string, key string) string {
	i, ok := t.cols[key]
	if !ok || i >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[i])
}

func (t importTable) has(key string) bool { _, ok := t.cols[key]; return ok }

// readImportTable finds the header row (exports may have a preamble) and maps known columns.
func readImportTable(r io.Reader) (importTable, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1
	cr.LazyQuotes = true
	records, err := cr.ReadAll()
	if err != nil {
		return importTable{}, invalidf("could not read CSV: %v", err)
	}
	for h := 0; h < len(records) && h < 20; h++ {
		cols := map[string]int{}
		for i, cell := range records[h] {
			norm := headerCleaner.ReplaceAllString(strings.ToLower(strings.TrimPrefix(cell, "\ufeff")), "")
			for key, aliases := range importColumns {
				if _, done := cols[key]; done {
					continue
				}
				for _, a := range aliases {
					if norm == a {
						cols[key] = i
						break
					}
				}
			}
		}
		if hasKey(cols, "qty") && (hasKey(cols, "symbol") || hasKey(cols, "name")) {
			rows := records[h+1:]
			if len(rows) > maxImportRows {
				return importTable{}, invalidf("file has more than %d rows", maxImportRows)
			}
			return importTable{cols: cols, rows: rows}, nil
		}
	}
	return importTable{}, invalidf("unrecognised CSV: expected a Zerodha/Groww holdings or tradebook export with symbol and quantity columns")
}

func hasKey(m map[string]int, k string) bool { _, ok := m[k]; return ok }

func parseImportNumber(s string) (float64, error) {
	s = strings.NewReplacer(",", "", "₹", "", " ", "").Replace(s)
	if s == "" {
		return 0, fmt.Errorf("missing number")
	}
	return strconv.ParseFloat(s, 64)
}

func parseImportDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	for _, layout := range importDateLayouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognised date %q", s)
}

func cleanSymbol(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	for _, suffix := range []string{"-EQ", "-BE", "-BZ", ".NS", ".BO"} {
		s = strings.TrimSuffix(s, suffix)
	}
	return s
}

func detectBroker(t importTable, requested string) string {
	requested = strings.ToLower(strings.TrimSpace(requested))
	if requested == "zerodha" || requested == "groww" {
		return requested
	}
	if t.has("name") && !t.has("symbol") || t.has("status") {
		return "groww"
	}
	return "zerodha"
}

type stockIndex struct {
	bySymbol map[string]*models.Stock
	byName   map[string]*models.Stock
}

func (ix stockIndex) find(symbol, name string) *models.Stock {
	if symbol != "" {
		if st := ix.bySymbol[symbol]; st != nil {
			return st
		}
	}
	return ix.byName[strings.ToLower(strings.TrimSpace(name))]
}

func (ix stockIndex) add(st *models.Stock) {
	ix.bySymbol[strings.ToUpper(st.Symbol)] = st
	if st.StockName != "" {
		ix.byName[strings.ToLower(st.StockName)] = st
	}
}

// Import reads a Zerodha/Groww holdings or tradebook CSV. Holdings create
// missing stocks with one buy at the average cost; tradebooks add each trade
// once (trade IDs are remembered so re-imports are skipped).
func (s *BrokerImportService) Import(ctx context.Context, userID string, r io.Reader, broker string, dryRun bool, asOf string) (*models.BrokerImportResult, error) {
	t, err := readImportTable(r)
	if err != nil {
		return nil, err
	}
	res := &models.BrokerImportResult{Broker: detectBroker(t, broker), DryRun: dryRun, Items: []models.BrokerImportRow{}}
	res.Kind = "holdings"
	if t.has("type") {
		res.Kind = "tradebook"
	}

	existing, err := s.repo.GetAll(ctx, userID)
	if err != nil {
		return nil, err
	}
	ix := stockIndex{bySymbol: map[string]*models.Stock{}, byName: map[string]*models.Stock{}}
	for i := range existing {
		ix.add(&existing[i])
	}
	touched := map[*models.Stock]bool{}
	created := map[*models.Stock]bool{}

	newStock := func(symbol, name, exchange string) *models.Stock {
		if symbol == "" {
			symbol = cleanSymbol(headerCleaner.ReplaceAllString(strings.ToLower(name), ""))
		}
		if name == "" {
			name = symbol
		}
		ex := models.ExchangeNSE
		if strings.EqualFold(exchange, "BSE") {
			ex = models.ExchangeBSE
		}
		st := &models.Stock{UserID: userID, StockName: name, Symbol: symbol, Exchange: ex, Transactions: []models.StockTransaction{}, Dividends: []models.StockDividend{}}
		ix.add(st)
		created[st] = true
		res.StocksCreated++
		return st
	}

	addErr := func(line int, sym, msg string) {
		res.Items = append(res.Items, models.BrokerImportRow{Symbol: sym, Action: "error", Message: msg})
		res.Errors = append(res.Errors, fmt.Sprintf("row %d: %s", line, msg))
	}

	if res.Kind == "holdings" {
		if !t.has("avg") {
			return nil, invalidf("holdings file needs an average price column")
		}
		date := time.Now().UTC().Truncate(24 * time.Hour)
		if asOf != "" {
			if date, err = parseDate(asOf); err != nil {
				return nil, err
			}
		}
		for i, row := range t.rows {
			line := i + 2
			symbol, name := cleanSymbol(t.get(row, "symbol")), t.get(row, "name")
			if symbol == "" && name == "" {
				continue
			}
			res.Rows++
			qtyF, err := parseImportNumber(t.get(row, "qty"))
			avg, err2 := parseImportNumber(t.get(row, "avg"))
			if err != nil || err2 != nil || qtyF <= 0 || avg <= 0 {
				addErr(line, symbol, "invalid quantity or average price")
				continue
			}
			if qtyF != math.Trunc(qtyF) {
				addErr(line, symbol, "fractional quantities are not supported")
				continue
			}
			qty := int(qtyF)
			item := models.BrokerImportRow{Symbol: symbol, Name: name, Quantity: qty, Price: round2(avg), Type: "buy", Date: date.Format("2006-01-02")}
			if st := ix.find(symbol, name); st != nil {
				item.Symbol = st.Symbol
				if st.TotalQuantity == qty {
					item.Action = "skip_exists"
					res.Skipped++
				} else {
					item.Action = "conflict"
					item.Message = fmt.Sprintf("already tracked with %d shares; adjust its transactions manually", st.TotalQuantity)
					res.Skipped++
				}
				res.Items = append(res.Items, item)
				continue
			}
			st := newStock(symbol, name, t.get(row, "exchange"))
			item.Symbol, item.Action = st.Symbol, "create"
			if symbol == "" {
				item.Message = "no symbol in file; check the NSE symbol so prices refresh"
			}
			st.Transactions = append(st.Transactions, models.StockTransaction{
				TransactionID: "import:" + res.Broker + ":holding:" + st.Symbol, Date: date, Type: models.StockBuy,
				Quantity: qty, PricePerShare: avg, Amount: round2(float64(qty) * avg),
			})
			touched[st] = true
			res.Items = append(res.Items, item)
		}
	} else {
		type trade struct {
			line   int
			st     *models.Stock
			txn    models.StockTransaction
			symbol string
		}
		var trades []trade
		for i, row := range t.rows {
			line := i + 2
			symbol, name := cleanSymbol(t.get(row, "symbol")), t.get(row, "name")
			if symbol == "" && name == "" {
				continue
			}
			if seg := strings.ToUpper(t.get(row, "segment")); seg != "" && seg != "EQ" {
				continue // F&O, currency, etc.
			}
			if status := strings.ToLower(t.get(row, "status")); status != "" && status != "executed" && status != "complete" && status != "completed" {
				continue
			}
			res.Rows++
			side := strings.ToLower(t.get(row, "type"))
			var typ models.StockTransactionType
			switch {
			case strings.HasPrefix(side, "b"):
				typ = models.StockBuy
			case strings.HasPrefix(side, "s"):
				typ = models.StockSell
			default:
				addErr(line, symbol, "trade type must be buy or sell")
				continue
			}
			qtyF, err := parseImportNumber(t.get(row, "qty"))
			if err != nil || qtyF <= 0 || qtyF != math.Trunc(qtyF) {
				addErr(line, symbol, "invalid quantity")
				continue
			}
			price, err := parseImportNumber(t.get(row, "price"))
			if err != nil || price <= 0 {
				if v, verr := parseImportNumber(t.get(row, "value")); verr == nil && v > 0 {
					price, err = v/qtyF, nil
				}
			}
			if err != nil || price <= 0 {
				addErr(line, symbol, "invalid price")
				continue
			}
			date, err := parseImportDate(t.get(row, "date"))
			if err != nil {
				addErr(line, symbol, err.Error())
				continue
			}
			tradeID := t.get(row, "tradeid")
			if tradeID == "" {
				h := sha1.Sum([]byte(strings.Join(row, "\x1f")))
				tradeID = "row-" + hex.EncodeToString(h[:8])
			}
			qty := int(qtyF)
			trades = append(trades, trade{line: line, symbol: symbol, txn: models.StockTransaction{
				TransactionID: "import:" + res.Broker + ":" + tradeID, Date: date, Type: typ,
				Quantity: qty, PricePerShare: round2(price), Amount: round2(float64(qty) * price),
			}})
			trades[len(trades)-1].st = ix.find(symbol, name)
			if trades[len(trades)-1].st == nil {
				trades[len(trades)-1].st = newStock(symbol, name, t.get(row, "exchange"))
			}
		}
		sort.SliceStable(trades, func(i, j int) bool { return trades[i].txn.Date.Before(trades[j].txn.Date) })
		for _, tr := range trades {
			item := models.BrokerImportRow{Symbol: tr.st.Symbol, Name: tr.st.StockName, Type: string(tr.txn.Type), Date: tr.txn.Date.Format("2006-01-02"), Quantity: tr.txn.Quantity, Price: tr.txn.PricePerShare}
			dup := false
			for _, existing := range tr.st.Transactions {
				if existing.TransactionID == tr.txn.TransactionID {
					dup = true
					break
				}
			}
			if dup {
				item.Action = "skip_duplicate"
				res.Skipped++
			} else {
				item.Action = "add_trade"
				tr.st.Transactions = append(tr.st.Transactions, tr.txn)
				touched[tr.st] = true
				res.TradesAdded++
			}
			res.Items = append(res.Items, item)
		}
	}

	// Drop stocks created only for rows that were all skipped or failed.
	for st := range created {
		if !touched[st] {
			res.StocksCreated--
		}
	}
	if dryRun {
		return res, nil
	}
	for st := range touched {
		sort.SliceStable(st.Transactions, func(i, j int) bool { return st.Transactions[i].Date.Before(st.Transactions[j].Date) })
		s.stocks.recalculate(st)
		if created[st] {
			if err := s.repo.Create(ctx, st); err != nil {
				return nil, err
			}
		} else if err := s.repo.Update(ctx, st); err != nil {
			return nil, err
		}
	}
	return res, nil
}
