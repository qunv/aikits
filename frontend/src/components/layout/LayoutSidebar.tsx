import type { MenuItemType } from 'antd/es/menu/interface';
import { LeftOutlined, MenuFoldOutlined, MenuUnfoldOutlined, MenuOutlined } from '@ant-design/icons';
import { SideMenu } from '@components/layout/SideMenu';
import { RepoSelector } from '@components/layout/RepoSelector';
import { Button, Grid, Layout } from 'antd';
import clsx from 'clsx';
import { observer } from 'mobx-react-lite';
import { useState } from 'react';
import { Outlet, useLocation, useNavigate } from 'react-router';
import { usePreferencesStore } from '@stores/StoreContext';
import { AnimateWrapper } from '@components/ui/AnimateWrapper';
import { Logo } from '@components/ui/Logo';
import GlobalSpinner from '@components/ui/GlobalSpinner';

const { Sider, Header, Content } = Layout;
const InnerLayout = Layout;
const { useBreakpoint } = Grid;

interface LayoutSidebarProps {
  menuItems: MenuItemType[];
  bottomMenuItems?: MenuItemType[];
  innerLayoutClassName?: string;
  backRoute?: string;
  backButtonText?: string;
}

export const LayoutSidebar = observer(
  ({ menuItems, bottomMenuItems, innerLayoutClassName, backRoute, backButtonText = 'Go back' }: LayoutSidebarProps) => {
    const navigate = useNavigate();
    const screens = useBreakpoint();
    const isMobile = !screens.lg;
    const location = useLocation();
    const preferencesStore = usePreferencesStore();
    const [mobileCollapsed, setMobileCollapsed] = useState(false);

    const collapsed = isMobile ? mobileCollapsed : preferencesStore.sidebarCollapsed;
    const setCollapsed = isMobile
      ? setMobileCollapsed
      : (val: boolean) => preferencesStore.setSidebarCollapsed(val);

    return (
      <Layout className="min-h-screen safe-area-top safe-area-bottom">
        <Sider
          breakpoint="lg"
          width={240}
          collapsedWidth={isMobile ? 0 : 80}
          collapsible
          collapsed={collapsed}
          trigger={null}
          className={clsx(
            'h-screen left-0 top-0 bottom-0 scrollbar-thin',
            isMobile ? 'fixed!' : 'sticky!',
          )}
          style={
            isMobile
              ? {
                  zIndex: 30,
                  left: !collapsed ? 0 : -240,
                  opacity: !collapsed ? 1 : 0,
                  transition: 'all 0.3s ease',
                }
              : { zIndex: 30 }
          }
          onCollapse={(value) => setCollapsed(value)}
        >
          <div className="flex flex-col h-full">
            {/* Logo */}
            <div className="flex items-center justify-center py-4 px-2">
              <Logo collapsed={collapsed} />
            </div>

            {backRoute && (
              <div className="p-4">
                <Button
                  color="default"
                  variant="filled"
                  icon={<LeftOutlined />}
                  block
                  onClick={() => navigate(backRoute)}
                >
                  {collapsed ? '' : backButtonText}
                </Button>
              </div>
            )}

            <SideMenu items={menuItems} collapsed={!isMobile && collapsed} />
            <div className="flex-1" />

            {bottomMenuItems && bottomMenuItems.length > 0 && (
              <SideMenu items={bottomMenuItems} collapsed={!isMobile && collapsed} />
            )}

            {!isMobile && (
              <div className="p-2 border-t border-gray-200">
                <Button
                  type="text"
                  block
                  icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
                  onClick={() => setCollapsed(!collapsed)}
                />
              </div>
            )}
          </div>
        </Sider>

        {isMobile && !collapsed && (
          <div
            onClick={() => setCollapsed(true)}
            style={{
              position: 'fixed',
              inset: 0,
              background: 'rgba(0,0,0,0.35)',
              zIndex: 25,
            }}
          />
        )}

        <InnerLayout className={innerLayoutClassName}>
          {/* Header lives inside the content column */}
          <Header className="sticky top-0 z-[100]! w-full flex items-center gap-2 px-4! lg:px-6!">
            {isMobile && (
              <Button
                type="text"
                icon={<MenuOutlined />}
                className="text-white!"
                onClick={() => setCollapsed(!collapsed)}
              />
            )}
            <RepoSelector />
            <GlobalSpinner />
          </Header>

          <Content className="p-4 overflow-y-auto">
            <AnimateWrapper>
              <Outlet key={location.pathname} />
            </AnimateWrapper>
          </Content>
        </InnerLayout>
      </Layout>
    );
  },
);
