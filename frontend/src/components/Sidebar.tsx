import { NavLink } from 'react-router-dom';

export default function Sidebar() {
  return (
    <aside className="sidebar">
      <h2>My Investments</h2>
      <nav>
        <NavLink to="/" end>Dashboard</NavLink>
        <NavLink to="/mutual-funds">Mutual Funds</NavLink>
        <NavLink to="/corporate-bonds">Corporate Bonds</NavLink>
        <NavLink to="/fixed-deposits">Fixed Deposits</NavLink>
        <NavLink to="/stocks">Stocks</NavLink>
        <NavLink to="/provident-fund">Provident Fund</NavLink>
      </nav>
    </aside>
  );
}
