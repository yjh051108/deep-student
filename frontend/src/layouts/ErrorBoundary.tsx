import { Component, ErrorInfo, ReactNode } from "react";

interface Props {
  children: ReactNode;
}

interface State {
  error: Error | null;
}

export class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo): void {
    // eslint-disable-next-line no-console
    console.error("[ErrorBoundary]", error, info);
  }

  private handleReload = () => {
    this.setState({ error: null });
  };

  render(): ReactNode {
    if (this.state.error) {
      return (
        <div className="flex h-full w-full items-center justify-center bg-slate-50 p-8">
          <div className="max-w-lg rounded-lg border border-rose-200 bg-white p-6 shadow">
            <h1 className="text-lg font-semibold text-rose-700">
              渲染出错了
            </h1>
            <p className="mt-2 text-sm text-slate-600">
              {this.state.error.message || "未知错误"}
            </p>
            <button
              type="button"
              onClick={this.handleReload}
              className="mt-4 rounded bg-rose-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-rose-500"
            >
              重试
            </button>
          </div>
        </div>
      );
    }
    return this.props.children;
  }
}
