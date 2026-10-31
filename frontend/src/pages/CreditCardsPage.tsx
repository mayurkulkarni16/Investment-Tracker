import { useEffect, useState } from 'react';
import { getCreditCards, createCreditCard, updateCreditCard, deleteCreditCard, addCardStatement, payStatement, addCardTransaction, addCardEMI, addCreditScore } from '../api/creditCards';
import type { CreditCard, CreateCreditCardRequest, UpdateCreditCardRequest, AddCardStatementRequest, PayStatementRequest, AddCardTransactionRequest, AddCardEMIRequest, AddCreditScoreRequest } from '../types';
import { formatCurrency } from '../utils/format';
import { useToast } from '../components/Toast';
import { useAuth } from '../context/AuthContext';
import { useConfirm } from '../components/ConfirmDialog';
import { Gauge, Pencil, Plus, Trash2 } from 'lucide-react';
import RowActions from '../components/RowActions';
import { PageSkeleton } from '../components/ui';

export default function CreditCardsPage() {
  const { isViewOnly, viewAsUserId } = useAuth();
  const [cards, setCards] = useState<CreditCard[]>([]);
  const [loading, setLoading] = useState(true);
  const [showAdd, setShowAdd] = useState(false);
  const [expanded, setExpanded] = useState<string | null>(null);
  const [search, setSearch] = useState('');
  const [showStatement, setShowStatement] = useState<string | null>(null);
  const [showPay, setShowPay] = useState<{ cardId: string; statementId: string } | null>(null);
  const [showEMI, setShowEMI] = useState<string | null>(null);
  const [showScore, setShowScore] = useState<string | null>(null);
  const [showEdit, setShowEdit] = useState<CreditCard | null>(null);
  const [showTxn, setShowTxn] = useState<{ cardId: string; statementId: string } | null>(null);
  const { toast } = useToast();
  const { confirm } = useConfirm();

  const [form, setForm] = useState<CreateCreditCardRequest>({
    card_name: '', bank_name: '', card_network: 'Visa', last_four_digits: '',
    card_holder_name: '', credit_limit: 0, billing_date: 1, due_date_offset: 20,
    joining_date: '',
  });
  const [stmtForm, setStmtForm] = useState<AddCardStatementRequest>({ month: '', statement_date: '', due_date: '', total_amount: 0, minimum_due: 0 });
  const [payForm, setPayForm] = useState<PayStatementRequest>({ statement_id: '', amount_paid: 0, paid_date: '' });
  const [emiForm, setEmiForm] = useState<AddCardEMIRequest>({ description: '', merchant_name: '', original_amount: 0, tenure_months: 3, start_date: '' });
  const [scoreForm, setScoreForm] = useState<AddCreditScoreRequest>({ date: '', score: 750, bureau: 'CIBIL' });
  const [editForm, setEditForm] = useState<UpdateCreditCardRequest>({});
  const [txnForm, setTxnForm] = useState<AddCardTransactionRequest>({ statement_id: '', date: '', description: '', amount: 0, category: '' });

  const load = () => { getCreditCards().then(r => setCards(r.data || [])).catch(() => {}).finally(() => setLoading(false)); };
  useEffect(() => { load(); }, [viewAsUserId]);

  const handleAdd = async (e: React.FormEvent) => { e.preventDefault(); try { await createCreditCard(form); setShowAdd(false); toast('Card added'); load(); } catch { toast('Failed to add card', 'error'); } };
  const handleDelete = async (id: string) => { if (!await confirm({ message: 'Delete this card?', danger: true, confirmLabel: 'Delete' })) return; try { await deleteCreditCard(id); toast('Card deleted'); load(); } catch { toast('Failed to delete', 'error'); } };
  const handleStatement = async (e: React.FormEvent) => { e.preventDefault(); if (!showStatement) return; try { await addCardStatement(showStatement, stmtForm); setShowStatement(null); toast('Statement added'); load(); } catch { toast('Failed to add statement', 'error'); } };
  const handlePay = async (e: React.FormEvent) => { e.preventDefault(); if (!showPay) return; try { await payStatement(showPay.cardId, payForm); setShowPay(null); toast('Payment recorded'); load(); } catch { toast('Failed to record payment', 'error'); } };
  const handleEMI = async (e: React.FormEvent) => { e.preventDefault(); if (!showEMI) return; try { await addCardEMI(showEMI, emiForm); setShowEMI(null); toast('EMI added'); load(); } catch { toast('Failed to add EMI', 'error'); } };
  const handleScore = async (e: React.FormEvent) => { e.preventDefault(); if (!showScore) return; try { await addCreditScore(showScore, scoreForm); setShowScore(null); toast('Credit score added'); load(); } catch { toast('Failed to add score', 'error'); } };
  const handleEdit = async (e: React.FormEvent) => { e.preventDefault(); if (!showEdit) return; try { await updateCreditCard(showEdit.id, editForm); setShowEdit(null); toast('Card updated'); load(); } catch { toast('Failed to update card', 'error'); } };
  const handleTxn = async (e: React.FormEvent) => { e.preventDefault(); if (!showTxn) return; try { await addCardTransaction(showTxn.cardId, txnForm); setShowTxn(null); toast('Transaction added'); load(); } catch { toast('Failed to add transaction', 'error'); } };

  const totalOutstanding = cards.reduce((s, c) => s + c.current_outstanding, 0);
  const totalLimit = cards.reduce((s, c) => s + c.credit_limit, 0);

  if (loading) return <PageSkeleton label="Loading" />;

  return (
    <div>
      <div className="page-header">
        <div>
          <h1>Credit Cards</h1>
          <p className="text-muted text-base">Outstanding: {formatCurrency(totalOutstanding)} / Limit: {formatCurrency(totalLimit)}</p>
        </div>
        {!isViewOnly && <button className="btn btn-primary" onClick={() => setShowAdd(true)}><Plus size={16} aria-hidden="true" />Add Card</button>}
      </div>

      {cards.length > 0 && (() => {
        const totalRewards = cards.reduce((s, c) => s + (c.reward_points || 0), 0);
        const utilization = totalLimit > 0 ? (totalOutstanding / totalLimit) * 100 : 0;
        return (
          <div className="card-grid stat-grid mb-5">
            <div className="stat-card">
              <div className="label">Total Cards</div>
              <div className="value">{cards.length}</div>
            </div>
            <div className="stat-card">
              <div className="label">Total Credit Limit</div>
              <div className="value">{formatCurrency(totalLimit)}</div>
            </div>
            <div className="stat-card">
              <div className="label">Outstanding</div>
              <div className={`value ${totalOutstanding > 0 ? 'negative' : ''}`}>{formatCurrency(totalOutstanding)}</div>
            </div>
            <div className="stat-card">
              <div className="label">Utilization / Rewards</div>
              <div className={`value ${utilization > 30 ? 'negative' : 'positive'}`}>{utilization.toFixed(1)}% | {totalRewards.toLocaleString()} pts</div>
            </div>
          </div>
        );
      })()}

      {cards.length === 0 ? (
        <div className="empty-state">
          <h3>No credit cards added yet</h3>
          <p>Add your first card to start tracking.</p>
          {!isViewOnly && <button className="btn btn-primary" onClick={() => setShowAdd(true)}><Plus size={16} aria-hidden="true" />Add Card</button>}
        </div>
      ) : (
        <div className="flex flex-col gap-4">
          <div className="search-bar mb-0">
            <input type="search" placeholder="Search by card name or bank..." value={search} onChange={e => setSearch(e.target.value)} />
          </div>
          {cards.filter(card => !search || card.card_name.toLowerCase().includes(search.toLowerCase()) || card.bank_name.toLowerCase().includes(search.toLowerCase())).map(card => (
            <div key={card.id} className="card">
              <div className="flex justify-between items-start cursor-pointer" onClick={() => setExpanded(expanded === card.id ? null : card.id)}>
                <div>
                  <h3 className="text-lg fw-semibold">{card.card_name} <span className="text-muted text-md">•••• {card.last_four_digits}</span></h3>
                  <p className="text-muted text-md">{card.bank_name} | {card.card_network}</p>
                </div>
                <div className="text-right">
                  <div className="text-2xl fw-bold">{formatCurrency(card.current_outstanding)}</div>
                  <div className="text-muted text-md">Limit: {formatCurrency(card.credit_limit)}</div>
                </div>
              </div>

              <div className="mt-2 flex gap-2 flex-wrap">
                <span className={`badge ${card.utilization_pct > 30 ? 'badge-matured' : 'badge-active'}`}>
                  Utilization: {card.utilization_pct?.toFixed(1)}%
                </span>
                {card.reward_points > 0 && <span className="badge badge-pending">Rewards: {card.reward_points.toLocaleString()} pts</span>}
                <span className="badge" style={{ background: 'var(--primary-soft)', color: 'var(--primary-text)' }}>Billing: {card.billing_date}th</span>
              </div>

              {card.credit_score_tips?.length > 0 && (
                <div className="mt-2 text-md" style={{ padding: '8px 12px', background: 'var(--primary-soft)', borderRadius: 6 }}>
                  <strong>Tips:</strong>
                  <ul className="ml-4 mt-1" style={{ listStyle: 'disc' }}>{card.credit_score_tips.map((t, i) => <li key={i}>{t}</li>)}</ul>
                </div>
              )}

              <div className="mt-3 flex gap-2 flex-wrap items-center" onClick={e => e.stopPropagation()}>
                {!isViewOnly && <button className="btn btn-sm btn-outline" onClick={() => setShowStatement(card.id)}><Plus size={14} aria-hidden="true" />Statement</button>}
                {!isViewOnly && (
                  <RowActions
                    label={`More actions for ${card.card_name}`}
                    actions={[
                      { label: 'Add EMI', icon: Plus, onClick: () => setShowEMI(card.id) },
                      { label: 'Record credit score', icon: Gauge, onClick: () => setShowScore(card.id) },
                      { label: 'Edit card', icon: Pencil, onClick: () => { setShowEdit(card); setEditForm({ card_name: card.card_name, credit_limit: card.credit_limit, billing_date: card.billing_date, due_date_offset: card.due_date_offset, reward_points: card.reward_points }); } },
                      { label: 'Delete card', icon: Trash2, onClick: () => handleDelete(card.id), danger: true },
                    ]}
                  />
                )}
              </div>

              {expanded === card.id && (
                <div className="mt-4">
                  {card.statements?.length > 0 && (
                    <div className="mb-4">
                      <h4 className="text-base fw-semibold mb-2">Statements</h4>
                      <table>
                        <thead><tr><th>Month</th><th className="text-right">Amount</th><th className="text-right">Paid</th><th>Status</th><th></th></tr></thead>
                        <tbody>
                          {card.statements.map(s => (
                            <tr key={s.statement_id}>
                              <td>{s.month}</td>
                              <td className="text-right">{formatCurrency(s.total_amount)}</td>
                              <td className="text-right">{formatCurrency(s.amount_paid)}</td>
                              <td><span className={`badge ${s.is_paid ? 'badge-active' : 'badge-matured'}`}>{s.is_paid ? (s.paid_full ? 'Full' : 'Partial') : 'Unpaid'}</span></td>
                              <td className="flex gap-1">
                                {!isViewOnly && !s.is_paid && <button className="btn btn-sm btn-primary" onClick={() => { setShowPay({ cardId: card.id, statementId: s.statement_id }); setPayForm({ statement_id: s.statement_id, amount_paid: s.total_amount, paid_date: '' }); }}>Pay</button>}
                                {!isViewOnly && <button className="btn btn-sm btn-outline" onClick={() => { setShowTxn({ cardId: card.id, statementId: s.statement_id }); setTxnForm({ statement_id: s.statement_id, date: '', description: '', amount: 0, category: '' }); }}><Plus size={14} aria-hidden="true" />Txn</button>}
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  )}

                  {card.card_emis?.length > 0 && (
                    <div className="mb-4">
                      <h4 className="text-base fw-semibold mb-2">Card EMIs</h4>
                      <table>
                        <thead><tr><th>Description</th><th className="text-right">EMI</th><th className="text-right">Remaining</th><th>Status</th></tr></thead>
                        <tbody>
                          {card.card_emis.map(e => (
                            <tr key={e.emi_id}>
                              <td>{e.description} ({e.merchant_name})</td>
                              <td className="text-right">{formatCurrency(e.emi_amount)}</td>
                              <td className="text-right">{e.remaining_months}/{e.tenure_months} months</td>
                              <td><span className={`badge ${e.status === 'active' ? 'badge-active' : 'badge-matured'}`}>{e.status}</span></td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  )}

                  {card.credit_scores?.length > 0 && (
                    <div>
                      <h4 className="text-base fw-semibold mb-2">Credit Score History</h4>
                      <div className="flex gap-3 flex-wrap">
                        {card.credit_scores.map(s => (
                          <div key={s.score_id} className="stat-card text-center p-3" style={{ minWidth: 100 }}>
                            <div className={`value ${s.score >= 750 ? 'positive' : s.score >= 650 ? '' : 'negative'}`} style={{ fontSize: 28 }}>{s.score}</div>
                            <div className="text-muted text-xs">{s.bureau} • {new Date(s.date).toLocaleDateString()}</div>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      {showAdd && (
        <div className="modal-overlay" onClick={() => setShowAdd(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Add Credit Card</h2>
            <form onSubmit={handleAdd}>
              <div className="form-row">
                <div className="form-group"><label>Card Name</label><input required value={form.card_name} onChange={e => setForm({ ...form, card_name: e.target.value })} /></div>
                <div className="form-group"><label>Bank Name</label><input required value={form.bank_name} onChange={e => setForm({ ...form, bank_name: e.target.value })} /></div>
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>Card Network</label>
                  <select value={form.card_network} onChange={e => setForm({ ...form, card_network: e.target.value })}>
                    <option>Visa</option><option>Mastercard</option><option>RuPay</option><option>Amex</option><option>Diners</option>
                  </select>
                </div>
                <div className="form-group"><label>Last 4 Digits</label><input maxLength={4} required value={form.last_four_digits} onChange={e => setForm({ ...form, last_four_digits: e.target.value })} /></div>
              </div>
              <div className="form-row">
                <div className="form-group"><label>Card Holder</label><input required value={form.card_holder_name} onChange={e => setForm({ ...form, card_holder_name: e.target.value })} /></div>
                <div className="form-group"><label>Credit Limit (₹)</label><input type="number" required value={form.credit_limit || ''} onChange={e => setForm({ ...form, credit_limit: +e.target.value })} /></div>
              </div>
              <div className="form-row-3">
                <div className="form-group"><label>Billing Date</label><input type="number" min={1} max={31} value={form.billing_date} onChange={e => setForm({ ...form, billing_date: +e.target.value })} /></div>
                <div className="form-group"><label>Due Offset (days)</label><input type="number" value={form.due_date_offset} onChange={e => setForm({ ...form, due_date_offset: +e.target.value })} /></div>
                <div className="form-group"><label>Annual Fee (₹)</label><input type="number" value={form.annual_fee || ''} onChange={e => setForm({ ...form, annual_fee: +e.target.value })} /></div>
              </div>
              <div className="form-group"><label>Joining Date</label><input type="date" required value={form.joining_date} onChange={e => setForm({ ...form, joining_date: e.target.value })} /></div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setShowAdd(false)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Add</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {showStatement && (
        <div className="modal-overlay" onClick={() => setShowStatement(null)}>
          <div className="modal modal-sm" onClick={e => e.stopPropagation()}>
            <h2>Add Statement</h2>
            <form onSubmit={handleStatement}>
              <div className="form-group"><label>Month (e.g. Jan-2025)</label><input required value={stmtForm.month} onChange={e => setStmtForm({ ...stmtForm, month: e.target.value })} /></div>
              <div className="form-row">
                <div className="form-group"><label>Statement Date</label><input type="date" required value={stmtForm.statement_date} onChange={e => setStmtForm({ ...stmtForm, statement_date: e.target.value })} /></div>
                <div className="form-group"><label>Due Date</label><input type="date" required value={stmtForm.due_date} onChange={e => setStmtForm({ ...stmtForm, due_date: e.target.value })} /></div>
              </div>
              <div className="form-row">
                <div className="form-group"><label>Total Amount (₹)</label><input type="number" min={0} placeholder="0" value={stmtForm.total_amount || ''} onChange={e => setStmtForm({ ...stmtForm, total_amount: +e.target.value || 0 })} /></div>
                <div className="form-group"><label>Minimum Due (₹)</label><input type="number" min={0} placeholder="0" value={stmtForm.minimum_due || ''} onChange={e => setStmtForm({ ...stmtForm, minimum_due: +e.target.value || 0 })} /></div>
              </div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setShowStatement(null)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Add</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {showPay && (
        <div className="modal-overlay" onClick={() => setShowPay(null)}>
          <div className="modal modal-sm" onClick={e => e.stopPropagation()}>
            <h2>Pay Statement</h2>
            <form onSubmit={handlePay}>
              <div className="form-group"><label>Amount Paid (₹)</label><input type="number" required value={payForm.amount_paid || ''} onChange={e => setPayForm({ ...payForm, amount_paid: +e.target.value })} /></div>
              <div className="form-group"><label>Paid Date</label><input type="date" required value={payForm.paid_date} onChange={e => setPayForm({ ...payForm, paid_date: e.target.value })} /></div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setShowPay(null)}>Cancel</button>
                <button type="submit" className="btn btn-success">Pay</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {showEMI && (
        <div className="modal-overlay" onClick={() => setShowEMI(null)}>
          <div className="modal modal-sm" onClick={e => e.stopPropagation()}>
            <h2>Add Card EMI</h2>
            <form onSubmit={handleEMI}>
              <div className="form-group"><label>Description</label><input required value={emiForm.description} onChange={e => setEmiForm({ ...emiForm, description: e.target.value })} /></div>
              <div className="form-group"><label>Merchant</label><input value={emiForm.merchant_name} onChange={e => setEmiForm({ ...emiForm, merchant_name: e.target.value })} /></div>
              <div className="form-row">
                <div className="form-group"><label>Original Amount (₹)</label><input type="number" required value={emiForm.original_amount || ''} onChange={e => setEmiForm({ ...emiForm, original_amount: +e.target.value })} /></div>
                <div className="form-group"><label>Tenure (months)</label><input type="number" required value={emiForm.tenure_months} onChange={e => setEmiForm({ ...emiForm, tenure_months: +e.target.value })} /></div>
              </div>
              <div className="form-row">
                <div className="form-group"><label>Interest Rate %</label><input type="number" step="0.1" value={emiForm.interest_rate || ''} onChange={e => setEmiForm({ ...emiForm, interest_rate: +e.target.value })} /></div>
                <div className="form-group"><label>Start Date</label><input type="date" required value={emiForm.start_date} onChange={e => setEmiForm({ ...emiForm, start_date: e.target.value })} /></div>
              </div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setShowEMI(null)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Add</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {showScore && (
        <div className="modal-overlay" onClick={() => setShowScore(null)}>
          <div className="modal modal-sm" onClick={e => e.stopPropagation()}>
            <h2>Add Credit Score</h2>
            <form onSubmit={handleScore}>
              <div className="form-group"><label>Date</label><input type="date" required value={scoreForm.date} onChange={e => setScoreForm({ ...scoreForm, date: e.target.value })} /></div>
              <div className="form-row">
                <div className="form-group"><label>Score (300-900)</label><input type="number" min={300} max={900} required value={scoreForm.score} onChange={e => setScoreForm({ ...scoreForm, score: +e.target.value })} /></div>
                <div className="form-group">
                  <label>Bureau</label>
                  <select value={scoreForm.bureau} onChange={e => setScoreForm({ ...scoreForm, bureau: e.target.value })}>
                    <option>CIBIL</option><option>Experian</option><option>Equifax</option><option>CRIF</option>
                  </select>
                </div>
              </div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setShowScore(null)}>Cancel</button>
                <button type="submit" className="btn btn-success">Add</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {showEdit && (
        <div className="modal-overlay" onClick={() => setShowEdit(null)}>
          <div className="modal modal-sm" onClick={e => e.stopPropagation()}>
            <h2>Edit Card</h2>
            <form onSubmit={handleEdit}>
              <div className="form-group"><label>Card Name</label><input value={editForm.card_name || ''} onChange={e => setEditForm({ ...editForm, card_name: e.target.value })} /></div>
              <div className="form-row">
                <div className="form-group"><label>Credit Limit (₹)</label><input type="number" value={editForm.credit_limit || ''} onChange={e => setEditForm({ ...editForm, credit_limit: +e.target.value })} /></div>
                <div className="form-group"><label>Reward Points</label><input type="number" value={editForm.reward_points || ''} onChange={e => setEditForm({ ...editForm, reward_points: +e.target.value })} /></div>
              </div>
              <div className="form-row">
                <div className="form-group"><label>Billing Date</label><input type="number" min={1} max={31} value={editForm.billing_date || ''} onChange={e => setEditForm({ ...editForm, billing_date: +e.target.value })} /></div>
                <div className="form-group"><label>Due Offset (days)</label><input type="number" value={editForm.due_date_offset || ''} onChange={e => setEditForm({ ...editForm, due_date_offset: +e.target.value })} /></div>
              </div>
              <div className="form-group"><label>Annual Fee (₹)</label><input type="number" value={editForm.annual_fee || ''} onChange={e => setEditForm({ ...editForm, annual_fee: +e.target.value })} /></div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setShowEdit(null)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Save</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {showTxn && (
        <div className="modal-overlay" onClick={() => setShowTxn(null)}>
          <div className="modal modal-sm" onClick={e => e.stopPropagation()}>
            <h2>Add Transaction</h2>
            <form onSubmit={handleTxn}>
              <div className="form-group"><label>Description</label><input required value={txnForm.description} onChange={e => setTxnForm({ ...txnForm, description: e.target.value })} /></div>
              <div className="form-row">
                <div className="form-group"><label>Date</label><input type="date" required value={txnForm.date} onChange={e => setTxnForm({ ...txnForm, date: e.target.value })} /></div>
                <div className="form-group"><label>Amount (₹)</label><input type="number" required value={txnForm.amount || ''} onChange={e => setTxnForm({ ...txnForm, amount: +e.target.value })} /></div>
              </div>
              <div className="form-group">
                <label>Category</label>
                <select value={txnForm.category} onChange={e => setTxnForm({ ...txnForm, category: e.target.value })}>
                  <option value="">Select</option>
                  <option value="food">Food & Dining</option><option value="shopping">Shopping</option><option value="travel">Travel</option>
                  <option value="fuel">Fuel</option><option value="groceries">Groceries</option><option value="entertainment">Entertainment</option>
                  <option value="utilities">Utilities</option><option value="health">Health</option><option value="other">Other</option>
                </select>
              </div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={() => setShowTxn(null)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Add</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
