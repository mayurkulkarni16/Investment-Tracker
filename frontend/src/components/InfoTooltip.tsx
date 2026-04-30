export default function InfoTooltip({ tip }: { tip: string }) {
  return <span className="info-tooltip" data-tip={tip}>?</span>;
}
