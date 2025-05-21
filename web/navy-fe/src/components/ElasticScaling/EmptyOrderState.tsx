import React from 'react';
import { Button, Empty } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import './EmptyOrderState.less';

interface EmptyOrderStateProps {
  onCreateOrder: () => void;
}

const EmptyOrderState: React.FC<EmptyOrderStateProps> = ({ onCreateOrder }) => {
  return (
    <div className="empty-order-state">
      <Empty
        image={Empty.PRESENTED_IMAGE_SIMPLE}
        description={
          <div>
            <h3>暂无订单</h3>
            <p>当前没有设备订单，您可以手动创建一个新订单</p>
          </div>
        }
      >
        <Button 
          type="primary" 
          icon={<PlusOutlined />} 
          onClick={onCreateOrder}
        >
          创建订单
        </Button>
      </Empty>
    </div>
  );
};

export default EmptyOrderState;
