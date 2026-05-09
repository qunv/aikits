import { Typography } from 'antd';

const { Title, Text } = Typography;

export default function HomePage() {
  return (
    <div className="flex flex-col gap-4">
      <Title level={3}>Welcome to Aikits</Title>
      <Text className="text-secondary">Select a tool from the sidebar to get started.</Text>
    </div>
  );
}
