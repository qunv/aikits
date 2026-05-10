import type { MenuItemType } from 'antd/es/menu/interface';
import { HomeOutlined, ApartmentOutlined, BookOutlined } from '@ant-design/icons';
import { LayoutSidebar } from '@components/layout/LayoutSidebar';
import { ROUTES } from '@constants/routes';
import { NavLink } from 'react-router';
import { SettingOutlined, UserOutlined } from '@ant-design/icons';

const menuItems: MenuItemType[] = [
  {
    key: ROUTES.home,
    icon: <HomeOutlined />,
    label: (
      <NavLink to={ROUTES.home} className="font-semibold">
        Home
      </NavLink>
    ),
  },
  {
    key: ROUTES.knowledgeGraph,
    icon: <ApartmentOutlined />,
    label: (
      <NavLink to={ROUTES.knowledgeGraph} className="font-semibold">
        Knowledge Graph
      </NavLink>
    ),
  },
  {
    key: ROUTES.features,
    icon: <BookOutlined />,
    label: (
      <NavLink to={ROUTES.features} className="font-semibold">
        Features
      </NavLink>
    ),
  },
];

const bottomMenuItems: MenuItemType[] = [
  {
    key: ROUTES.settings,
    icon: <SettingOutlined />,
    label: (
      <NavLink to={ROUTES.settings} className="font-semibold">
        Settings
      </NavLink>
    ),
  },
  {
    key: '/me',
    icon: <UserOutlined />,
    label: (
      <NavLink to="/me" className="font-semibold">
        Profile
      </NavLink>
    ),
  },
];

function AdminLayout() {
  return (
    <LayoutSidebar
      menuItems={menuItems}
      bottomMenuItems={bottomMenuItems}
      innerLayoutClassName="rounded-t-lg!"
    />
  );
}

export default AdminLayout;
