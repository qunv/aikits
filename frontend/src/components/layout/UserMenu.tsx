import { LogoutOutlined } from '@ant-design/icons';
import { useUserStore } from '@stores/StoreContext';
import { getInitials } from '@utils/name_helper';
import { Avatar, Dropdown } from 'antd';
import { observer } from 'mobx-react-lite';

export const UserMenu = observer(() => {
  const userStore = useUserStore();
  const displayName = userStore.displayName || 'User';
  const initials = getInitials(displayName);

  const menuItems = [
    {
      key: 'display-name',
      disabled: true,
      label: (
        <div className="min-w-[180px] rounded-xl bg-slate-50 px-3 py-2">
          <div className="truncate text-sm font-semibold text-slate-700">{displayName}</div>
        </div>
      ),
    },
    { type: 'divider' as const },
    {
      key: 'logout',
      icon: <LogoutOutlined />,
      label: 'Quit',
      danger: true,
      onClick: () => window.close(),
    },
  ];

  return (
    <Dropdown
      menu={{ items: menuItems, style: { width: 200 } }}
      trigger={['click']}
      placement="bottomRight"
    >
      <Avatar
        className="bg-emerald-100! text-emerald-900! shadow-[0_4px_12px_rgba(0,0,0,0.1)] cursor-pointer"
        style={{ fontSize: 14, fontWeight: 600 }}
      >
        {initials}
      </Avatar>
    </Dropdown>
  );
});
