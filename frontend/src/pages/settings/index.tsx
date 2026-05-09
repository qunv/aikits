import { Typography } from 'antd';

const { Title } = Typography;

export default function SettingsPage() {
  return (
    <div className="flex flex-col gap-4">
      <Title level={3}>Settings</Title>
    </div>
  );
}
