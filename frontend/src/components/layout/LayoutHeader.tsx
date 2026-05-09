import { MenuOutlined } from '@ant-design/icons';
import GlobalSpinner from '@components/ui/GlobalSpinner';
import { Logo } from '@components/ui/Logo';
import { Button, Layout } from 'antd';

const { Header } = Layout;

interface LayoutHeaderProps {
  onMenuToggle: () => void;
}

export function LayoutHeader({ onMenuToggle }: LayoutHeaderProps) {
  return (
    <Header className="sticky top-0 z-[100]! w-full flex items-center flex-row justify-between px-4! lg:px-8!">
      <div className="flex items-center gap-2">
        <Button
          type="text"
          icon={<MenuOutlined />}
          className="text-white! lg:hidden!"
          onClick={onMenuToggle}
        />
        <Logo invert />
        <GlobalSpinner />
      </div>
    </Header>
  );
}
