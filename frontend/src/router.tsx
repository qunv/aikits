import AdminLayout from '@components/layout/LayoutAdmin';
import { ROUTES } from '@constants/routes';
import { ErrorBoundary, RootLayout } from '@/RootLayout';
import { lazy } from 'react';
import { createHashRouter, redirect } from 'react-router';

const HomePage = lazy(() => import('@pages/home'));
const SettingsPage = lazy(() => import('@pages/settings'));
const KnowledgeGraphPage = lazy(() => import('@pages/knowledge-graph'));
const FeaturesPage = lazy(() => import('@pages/features'));
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
          { path: ROUTES.knowledgeGraph, Component: KnowledgeGraphPage },
          { path: ROUTES.features, Component: FeaturesPage },
        ],
      },
      { path: '*', Component: NotFoundPage },
    ],
  },
]);
