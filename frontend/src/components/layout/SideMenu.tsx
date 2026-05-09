import type { MenuItemType } from 'antd/es/menu/interface';
import { Menu } from 'antd';
import { useSelectedMenuKey } from '@hooks/useSelectedMenuKey';

interface SideMenuProps {
  items?: MenuItemType[];
  collapsed?: boolean;
}

export function SideMenu({ items, collapsed }: SideMenuProps) {
  const selectedKeys = useSelectedMenuKey(items || []);

  return (
    <Menu
      mode="inline"
      inlineCollapsed={collapsed}
      selectedKeys={[selectedKeys]}
      items={items}
      className="border-none! p-2!"
    />
  );
}
