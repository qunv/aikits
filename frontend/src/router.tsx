import AdminLayout from '@components/layout/LayoutAdmin';
import { ROUTES } from '@constants/routes';
import { ErrorBoundary, RootLayout } from '@/RootLayout';
import { lazy } from 'react';
import { createHashRouter, redirect } from 'react-router';

const HomePage = lazy(() => import('@pages/home'));
const SettingsPage = lazy(() => import('@pages/settings'));
const NotFoundPage = lazy(() => import('@pages/not-found'));

export const router = createHashRouter([
  {
    Component: RootLayout,
    errorElement: <ErrorBoundary />,
    children: [
      { index: true, loader: () => redirect(ROUTES.home) },
      {
        Component: AdminLayout,
        children: [
          { path: ROUTES.home, Component: HomePage },
          { path: ROUTES.settings, Component: SettingsPage },
        ],
      },
      { path: '*', Component: NotFoundPage },
    ],
  },
]);
