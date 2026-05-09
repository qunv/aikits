import { createContext, useCallback, useContext, useMemo, useState } from 'react';
import { LoadingOverlay } from '@components/ui/LoadingOverlay';

interface LoadingContextValue {
  isLoading: boolean;
  showLoading: () => void;
  hideLoading: () => void;
  setLoading: (value: boolean) => void;
  withLoading: <T>(fn: () => Promise<T>) => Promise<T>;
}

const LoadingContext = createContext<LoadingContextValue | null>(null);

export function LoadingProvider({ children }: { children: React.ReactNode }) {
  const [isLoading, setIsLoading] = useState(false);

  const showLoading = useCallback(() => setIsLoading(true), []);
  const hideLoading = useCallback(() => setIsLoading(false), []);
  const setLoading = useCallback((value: boolean) => setIsLoading(value), []);

  const withLoading = useCallback(async <T,>(fn: () => Promise<T>): Promise<T> => {
    setIsLoading(true);
    try {
      return await fn();
    } finally {
      setIsLoading(false);
    }
  }, []);

  const value = useMemo(
    () => ({ isLoading, showLoading, hideLoading, setLoading, withLoading }),
    [isLoading, showLoading, hideLoading, setLoading, withLoading],
  );

  return (
    <LoadingContext.Provider value={value}>
      {children}
      <LoadingOverlay open={isLoading} tip="Loading..." />
    </LoadingContext.Provider>
  );
}

export function useLoading() {
  const context = useContext(LoadingContext);
  if (!context) {
    throw new Error('useLoading must be used within LoadingProvider');
  }
  return context;
}
