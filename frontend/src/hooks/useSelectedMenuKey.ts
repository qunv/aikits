import type { MenuItemType } from 'antd/es/menu/interface';
import { ROUTES } from '@constants/routes';
import { useLocation } from 'react-router';

export function useSelectedMenuKey(menuItems: MenuItemType[]) {
  const { pathname } = useLocation();

  for (let i = 0; i < menuItems.length; i++) {
    const item = menuItems[i];
    if (pathname.startsWith(item.key as string)) {
      return item.key as string;
    }
  }

  return pathname;
}
