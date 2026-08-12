import React, { Component, ErrorInfo, ReactNode } from 'react';
import { AlertCircle, RefreshCw, Home, DownloadCloud } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { isChunkLoadError } from '@/lib/chunk-reload';

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
}

interface State {
  hasError: boolean;
  error: Error | null;
  errorInfo: ErrorInfo | null;
  errorCount: number;
}

export class ErrorBoundary extends Component<Props, State> {
  private recoveryTimeoutId: ReturnType<typeof setTimeout> | null = null;

  constructor(props: Props) {
    super(props);
    this.state = {
      hasError: false,
      error: null,
      errorInfo: null,
      errorCount: 0,
    };
  }

  static getDerivedStateFromError(error: Error): Partial<State> {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    // Special handling for DOM manipulation errors
    if (error.message.includes('removeChild') || error.message.includes('insertBefore') || error.message.includes('appendChild')) {
      console.warn('DOM manipulation error caught, attempting to recover:', error.message);

      // Small delay to allow DOM to stabilize before re-rendering
      this.recoveryTimeoutId = setTimeout(() => {
        this.setState(prevState => ({
          errorInfo,
          errorCount: prevState.errorCount + 1,
        }));
      }, 50);

      return;
    }

    // Log error to console in development
    if (process.env.NODE_ENV === 'development') {
      console.error('ErrorBoundary caught an error:', error, errorInfo);
    }

    // Call custom error handler if provided
    if (this.props.onError) {
      this.props.onError(error, errorInfo);
    }

    // Update state with error details
    this.setState(prevState => ({
      errorInfo,
      errorCount: prevState.errorCount + 1,
    }));

    // In production, you might want to log to an error reporting service
    // Example: logErrorToService(error, errorInfo);
  }

  componentWillUnmount() {
    if (this.recoveryTimeoutId) {
      clearTimeout(this.recoveryTimeoutId);
      this.recoveryTimeoutId = null;
    }
  }

  handleReset = () => {
    this.setState({
      hasError: false,
      error: null,
      errorInfo: null,
    });
  };

  handleGoHome = () => {
    window.location.href = '/';
  };

  /**
   * Hard reload to pick up fresh chunk hashes after a deploy. This is the
   * manual escape hatch — it bypasses the sessionStorage loop guard in
   * chunk-reload.ts because the user explicitly asked to reload.
   */
  handleHardReload = () => {
    window.location.reload();
  };

  render() {
    if (this.state.hasError) {
      // Use custom fallback if provided
      if (this.props.fallback) {
        return <>{this.props.fallback}</>;
      }

      // Dedicated UI for stale-chunk failures: a deploy removed a hashed JS
      // chunk the browser was still referencing. A hard reload fixes it.
      if (isChunkLoadError(this.state.error)) {
        return (
          <div className="min-h-screen flex items-center justify-center bg-background p-4">
            <Card className="max-w-md w-full">
              <CardHeader>
                <div className="flex items-center gap-2">
                  <DownloadCloud className="h-6 w-6 text-primary" />
                  <CardTitle>Đang cập nhật ứng dụng</CardTitle>
                </div>
                <CardDescription>
                  Chúng tôi vừa triển khai phiên bản mới. Vui lòng tải lại trang
                  để tiếp tục.
                </CardDescription>
              </CardHeader>
              <CardFooter>
                <Button onClick={this.handleHardReload} className="w-full bg-primary text-primary-foreground hover:bg-primary/90">
                  <RefreshCw className="mr-2 h-4 w-4" />
                  Tải lại trang
                </Button>
              </CardFooter>
            </Card>
          </div>
        );
      }

      // Default error UI
      return (
        <div className="min-h-screen flex items-center justify-center bg-background p-4">
          <Card className="max-w-2xl w-full">
            <CardHeader>
              <div className="flex items-center gap-2">
                <AlertCircle className="h-6 w-6 text-destructive" />
                <CardTitle>Đã xảy ra lỗi không mong muốn</CardTitle>
              </div>
              <CardDescription>
                Chúng tôi xin lỗi vì sự bất tiện này. Vui lòng thử lại hoặc liên hệ với bộ phận hỗ trợ nếu lỗi vẫn tiếp tục.
              </CardDescription>
            </CardHeader>

            <CardContent className="space-y-4">
              {/* Error message */}
              <Alert variant="destructive">
                <AlertCircle className="h-4 w-4" />
                <AlertTitle>Chi tiết lỗi</AlertTitle>
                <AlertDescription className="mt-2">
                  <p className="font-mono typography-body-medium">
                    {this.state.error?.message || 'Lỗi không xác định'}
                  </p>
                </AlertDescription>
              </Alert>

              {/* Stack trace in development */}
              {process.env.NODE_ENV === 'development' && this.state.errorInfo && (
                <details className="cursor-pointer">
                  <summary className="typography-body-medium typography-label-medium text-muted-foreground hover:text-foreground">
                    Xem chi tiết kỹ thuật (Development only)
                  </summary>
                  <pre className="mt-2 p-3 bg-muted rounded-xl typography-body-small overflow-auto max-h-64">
                    {this.state.errorInfo.componentStack}
                  </pre>
                </details>
              )}

              {/* Error count warning */}
              {this.state.errorCount > 2 && (
                <Alert>
                  <AlertCircle className="h-4 w-4" />
                  <AlertDescription>
                    Lỗi này đã xảy ra {this.state.errorCount} lần.
                    Vui lòng làm mới trang hoặc xóa bộ nhớ cache của trình duyệt.
                  </AlertDescription>
                </Alert>
              )}
            </CardContent>

            <CardFooter className="flex gap-3">
              <Button onClick={this.handleReset} variant="ghost" className="flex-1 bg-primary text-primary-foreground hover:bg-primary/90">
                <RefreshCw className="mr-2 h-4 w-4" />
                Thử lại
              </Button>
              <Button onClick={this.handleGoHome} variant="outline" className="flex-1">
                <Home className="mr-2 h-4 w-4" />
                Về trang chủ
              </Button>
            </CardFooter>
          </Card>
        </div>
      );
    }

    return this.props.children;
  }
}

/**
 * Hook for using error boundary
 */
export const useErrorHandler = () => {
  const [error, setError] = React.useState<Error | null>(null);

  React.useEffect(() => {
    if (error) {
      throw error;
    }
  }, [error]);

  return setError;
};

/**
 * Component error boundary wrapper with custom UI
 */
export const ComponentErrorBoundary: React.FC<{
  children: ReactNode;
  componentName?: string;
}> = ({ children, componentName }) => {
  return (
    <ErrorBoundary
      fallback={
        <div className="p-6 border border-destructive/20 rounded-xl bg-destructive/5">
          <div className="flex items-center gap-2 text-destructive">
            <AlertCircle className="h-5 w-5" />
            <p className="font-medium">
              Lỗi khi tải {componentName || 'component'}
            </p>
          </div>
          <p className="typography-body-medium text-muted-foreground mt-2">
            Vui lòng làm mới trang để thử lại
          </p>
        </div>
      }
    >
      {children}
    </ErrorBoundary>
  );
};

/**
 * API error boundary for handling API failures
 */
export const ApiErrorBoundary: React.FC<{
  children: ReactNode;
  onRetry?: () => void;
}> = ({ children, onRetry }) => {
  return (
    <ErrorBoundary
      onError={(error) => {
        if (error.message.includes('API') || error.message.includes('Network')) {
          console.error('API Error:', error);
        }
      }}
      fallback={
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertTitle>Lỗi kết nối</AlertTitle>
          <AlertDescription className="mt-2">
            <p>Không thể kết nối đến máy chủ. Vui lòng kiểm tra kết nối mạng và thử lại.</p>
            {onRetry && (
              <Button onClick={onRetry} variant="outline" size="sm" className="mt-3">
                <RefreshCw className="mr-2 h-4 w-4" />
                Thử lại
              </Button>
            )}
          </AlertDescription>
        </Alert>
      }
    >
      {children}
    </ErrorBoundary>
  );
};

/**
 * Section-scoped error boundary for layout sections.
 * Shows a compact error with retry — isolates feature crashes from the rest of the app.
 */
export class SectionErrorBoundary extends Component<
  { children: ReactNode; sectionName?: string },
  { hasError: boolean; error: Error | null }
> {
  constructor(props: { children: ReactNode; sectionName?: string }) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error) {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error(`SectionErrorBoundary [${this.props.sectionName || 'unknown'}]:`, error, errorInfo);
  }

  handleRetry = () => {
    this.setState({ hasError: false, error: null });
  };

  render() {
    if (this.state.hasError) {
      return (
        <div className="flex flex-col items-center justify-center gap-4 p-8 text-center">
          <div className="rounded-full bg-destructive/10 p-3">
            <AlertCircle className="h-6 w-6 text-destructive" />
          </div>
          <div>
            <p className="font-medium">
              Lỗi khi tải{this.props.sectionName ? ` ${this.props.sectionName}` : ' nội dung'}
            </p>
            <p className="text-sm text-muted-foreground mt-1">
              {this.state.error?.message || 'Đã xảy ra lỗi không xác định'}
            </p>
          </div>
          <Button onClick={this.handleRetry} variant="outline" size="sm">
            <RefreshCw className="mr-2 h-4 w-4" />
            Thử lại
          </Button>
        </div>
      );
    }
    return this.props.children;
  }
}
