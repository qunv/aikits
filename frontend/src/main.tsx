import './wailsMock';
import { createRoot } from 'react-dom/client';
import { RouterProvider } from 'react-router';
import { router } from './router';
import './app.css';

createRoot(document.getElementById('root')!).render(<RouterProvider router={router} />);

