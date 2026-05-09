import { LoadingOutlined } from '@ant-design/icons';
import { useIsFetching } from '@tanstack/react-query';
import { Spin } from 'antd';
import { useNavigation } from 'react-router';

function GlobalSpinner() {
  const navigation = useNavigation();
  const isFetching = useIsFetching();
  const isLoading = navigation.state === 'loading';

  if (!isLoading && !isFetching) return null;

  return <Spin indicator={<LoadingOutlined spin className="text-white!" />} />;
}

export default GlobalSpinner;
