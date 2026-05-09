import { CustomSpin } from '@components/ui/CustomSpin';
import { LoadingProvider } from '@providers/LoadingContext';
import { QueryProvider } from '@providers/QueryProvider';
import { ThemeProvider } from '@providers/ThemeProvider';
import { StoreProvider } from '@stores/StoreContext';
import { App as AntApp } from 'antd';
import { Suspense } from 'react';
import { isRouteErrorResponse, Outlet, useRouteError } from 'react-router';

export function RootLayout() {
  return (
    <StoreProvider>
      <QueryProvider>
        <ThemeProvider>
          <AntApp>
            <LoadingProvider>
              <Suspense fallback={<CustomSpin size="large" />}>
                <Outlet />
              </Suspense>
            </LoadingProvider>
          </AntApp>
        </ThemeProvider>
      </QueryProvider>
    </StoreProvider>
  );
}

export function ErrorBoundary() {
  const error = useRouteError();
  let message = 'Oops!';
  let details = 'An unexpected error occurred.';
  let stack: string | undefined;

  if (isRouteErrorResponse(error)) {
    message = error.status === 404 ? '404' : 'Error';
    details =
      error.status === 404
        ? 'The requested page could not be found.'
        : error.statusText || details;
  } else if (import.meta.env.DEV && error instanceof Error) {
    details = error.message;
    stack = error.stack;
  }

  return (
    <main className="pt-16 p-4 container mx-auto">
      <h1>{message}</h1>
      <p>{details}</p>
      {stack && (
        <pre className="w-full p-4 overflow-x-auto">
          <code>{stack}</code>
        </pre>
      )}
    </main>
  );
}
