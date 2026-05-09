import { ConfigProvider } from 'antd';

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  return (
    <ConfigProvider
      theme={{
        token: {
          borderRadius: 12,
          colorPrimary: 'rgb(42 130 152)',
          colorPrimaryActive: 'rgb(26 44 69)',
          colorSuccess: '#389e0d',
          colorWarning: '#d48806',
          colorTextDescription: 'rgba(0, 0, 0, 0.65)',
        },
        components: {
          Layout: {
            headerHeight: 56,
            headerBg: 'rgb(26 44 69)',
            siderBg: '#ebebeb',
            bodyBg: '#f1f1f1',
          },
          Menu: {
            itemBg: '#ebebeb',
            subMenuItemBg: '#ebebeb',
            itemHoverBg: '#f1f1f1',
            itemSelectedBg: '#ffffff',
            itemSelectedColor: '#000000',
            itemBorderRadius: 12,
          },
          Button: {
            borderRadius: 8,
            borderRadiusLG: 8,
            fontWeight: 500,
            contentFontSizeSM: 12,
          },
          Card: {
            boxShadow: '0 4px 12px rgba(0, 0, 0, 0.5)',
          },
          Input: {
            borderRadius: 8,
          },
        },
      }}
    >
      <div
        style={
          {
            '--app-gradient-primary': 'linear-gradient(180deg, #1E3A8A 0%, #312E81 50%, #0F172A 100%)',
          } as React.CSSProperties
        }
      >
        {children}
      </div>
    </ConfigProvider>
  );
}
