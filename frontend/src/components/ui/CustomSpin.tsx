import type { SpinProps } from 'antd';
import { LoadingOutlined } from '@ant-design/icons';
import { Spin } from 'antd';

export function CustomSpin(props: SpinProps) {
  return <Spin indicator={<LoadingOutlined spin />} {...props} />;
}
