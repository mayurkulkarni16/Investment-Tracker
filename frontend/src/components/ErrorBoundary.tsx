import { Component, type ErrorInfo, type ReactNode } from 'react';

interface Props {
  children: ReactNode;
}

interface State {
  error: Error | null;
}

export default class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('Page crashed:', error, info.componentStack);
  }

  render() {
    if (!this.state.error) return this.props.children;
    return (
      <div className="card empty-state" role="alert">
        <h3>Something went wrong on this page</h3>
        <p>The data may be in an unexpected format. Try again, or open another page from the menu.</p>
        <button className="btn btn-primary" onClick={() => this.setState({ error: null })}>Try again</button>
      </div>
    );
  }
}
