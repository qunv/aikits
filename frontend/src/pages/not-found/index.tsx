import { Button, Result } from 'antd';
import { useNavigate } from 'react-router';
import { ROUTES } from '@constants/routes';

export default function NotFoundPage() {
  const navigate = useNavigate();
  return (
    <Result
      status="404"
      title="404"
      subTitle="Sorry, the page you visited does not exist."
      extra={<Button type="primary" onClick={() => navigate(ROUTES.home)}>Back Home</Button>}
    />
  );
}
