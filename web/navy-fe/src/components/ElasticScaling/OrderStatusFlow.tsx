import React from 'react';
import { Steps } from 'antd';
import { 
  CheckCircleOutlined, 
  LoadingOutlined, 
  PlayCircleOutlined,
  RocketOutlined,
  SettingOutlined,
  CheckOutlined,
  SyncOutlined,
  PauseCircleOutlined,
  CloseCircleOutlined
} from '@ant-design/icons';
import { OrderStatus, isFinalStatus } from '../../types/order';

interface OrderStatusFlowProps {
  actionType: 'pool_entry' | 'pool_exit' | 'maintenance_request' | 'maintenance_uncordon';
  currentStatus: OrderStatus;
}

const OrderStatusFlow: React.FC<OrderStatusFlowProps> = ({ actionType, currentStatus }) => {
  const getStepStatus = (stepStatus: OrderStatus) => {
    if (currentStatus === stepStatus) {
      return 'process';
    }
    
    // 如果当前状态是已取消，且步骤是取消状态，则显示为完成
    if (currentStatus === 'cancelled' && stepStatus === 'cancelled') {
      return 'finish';
    }
    
    // 如果当前状态是已取消，且步骤不是取消状态，则根据步骤顺序判断
    if (currentStatus === 'cancelled') {
      const statusOrder = {
        'pending': 0,
        'processing': 1,
        'returning': 2,
        'return_completed': 3,
        'no_return': 3,
        'completed': 4,
        'failed': -1,
        'cancelled': 1,
        'ignored': -1,
        'pending_confirmation': 0,
        'scheduled_for_maintenance': 1,
        'maintenance_in_progress': 2
      };
      const stepOrder = statusOrder[stepStatus] || 0;
      if (stepOrder < 1) {
        return 'finish';
      }
      return 'wait';
    }
    
    const statusOrder = {
      'pending': 0,
      'processing': 1,
      'returning': 2,
      'return_completed': 3,
      'no_return': 3,
      'completed': 4,
      'failed': -1,
      'cancelled': 1,
      'ignored': -1,
      'pending_confirmation': 0,
      'scheduled_for_maintenance': 1,
      'maintenance_in_progress': 2
    };
    
    const currentOrder = statusOrder[currentStatus] || 0;
    const stepOrder = statusOrder[stepStatus] || 0;
    
    if (currentOrder > stepOrder) {
      return 'finish';
    }
    
    return 'wait';
  };



  // 入池订单流程
  const poolEntrySteps = [
    {
      title: '创建',
      status: 'pending' as OrderStatus,
      description: '订单已创建，等待处理'
    },
    {
      title: '处理中',
      status: 'processing' as OrderStatus,
      description: '正在执行入池操作'
    },
    {
      title: '完成',
      status: 'completed' as OrderStatus,
      description: '入池操作已完成'
    }
  ];

  // 入池订单取消流程
  const poolEntryCancelledSteps = [
    {
      title: '创建',
      status: 'pending' as OrderStatus,
      description: '订单已创建，等待处理'
    },
    {
      title: '已取消',
      status: 'cancelled' as OrderStatus,
      description: '订单已被取消'
    }
  ];

  // 退池订单流程（归还路径）
  const poolExitReturnSteps = [
    {
      title: '创建',
      status: 'pending' as OrderStatus,
      description: '订单已创建，等待处理'
    },
    {
      title: '处理中',
      status: 'processing' as OrderStatus,
      description: '正在执行退池操作'
    },
    {
      title: '归还中',
      status: 'returning' as OrderStatus,
      description: '设备正在归还中'
    },
    {
      title: '归还完成',
      status: 'return_completed' as OrderStatus,
      description: '设备归还完成'
    }
  ];

  // 退池订单流程（无需归还路径）
  const poolExitNoReturnSteps = [
    {
      title: '创建',
      status: 'pending' as OrderStatus,
      description: '订单已创建，等待处理'
    },
    {
      title: '处理中',
      status: 'processing' as OrderStatus,
      description: '正在执行退池操作'
    },
    {
      title: '归还中',
      status: 'returning' as OrderStatus,
      description: '设备正在归还中'
    },
    {
      title: '无需归还',
      status: 'no_return' as OrderStatus,
      description: '确认无需归还设备'
    }
  ];

  // 退池订单取消流程
  const poolExitCancelledSteps = [
    {
      title: '创建',
      status: 'pending' as OrderStatus,
      description: '订单已创建，等待处理'
    },
    {
      title: '已取消',
      status: 'cancelled' as OrderStatus,
      description: '订单已被取消'
    }
  ];

  // 维护订单取消流程
  const maintenanceCancelledSteps = [
    {
      title: '创建',
      status: 'pending' as OrderStatus,
      description: '订单已创建，等待处理'
    },
    {
      title: '已取消',
      status: 'cancelled' as OrderStatus,
      description: '订单已被取消'
    }
  ];

  const getSteps = () => {
    // 如果当前状态是已取消，显示取消流程
    if (currentStatus === 'cancelled') {
      if (actionType === 'pool_entry') {
        return poolEntryCancelledSteps;
      }
      if (actionType === 'pool_exit') {
        return poolExitCancelledSteps;
      }
      // 维护类型订单
      return maintenanceCancelledSteps;
    }
    
    if (actionType === 'pool_entry') {
      return poolEntrySteps;
    }
    
    if (actionType === 'pool_exit') {
      // 对于退池订单，根据当前状态判断走哪个流程
      if (currentStatus === 'no_return') {
        return poolExitNoReturnSteps;
      }
      return poolExitReturnSteps;
    }
    
    // 对于维护类型的订单，使用简单的流程
    return poolEntrySteps;
  };

  const steps = getSteps();
  const currentStepIndex = steps.findIndex(step => step.status === currentStatus);

  const containerStyle: React.CSSProperties = {
    padding: '24px',
    background: 'linear-gradient(135deg, #f8fafc 0%, #f1f5f9 100%)',
    borderRadius: '12px',
    boxShadow: '0 1px 3px 0 rgba(0, 0, 0, 0.1), 0 1px 2px 0 rgba(0, 0, 0, 0.06)',
    position: 'relative',
    overflow: 'hidden'
  };

  const titleStyle: React.CSSProperties = {
    fontSize: '16px',
    fontWeight: 600,
    color: '#1e293b',
    marginBottom: '20px',
    display: 'flex',
    alignItems: 'center',
    gap: '8px'
  };

  const getActionIcon = () => {
    switch (actionType) {
      case 'pool_entry':
        return <PlayCircleOutlined style={{ color: '#10b981' }} />;
      case 'pool_exit':
        return <PlayCircleOutlined style={{ color: '#f59e0b', transform: 'rotate(180deg)' }} />;
      case 'maintenance_request':
      case 'maintenance_uncordon':
        return <PlayCircleOutlined style={{ color: '#8b5cf6' }} />;
      default:
        return <PlayCircleOutlined style={{ color: '#6b7280' }} />;
    }
  };

  const getActionTitle = () => {
    switch (actionType) {
      case 'pool_entry':
        return '入池流程';
      case 'pool_exit':
        return '退池流程';
      case 'maintenance_request':
        return '维护申请流程';
      case 'maintenance_uncordon':
        return '维护解除流程';
      default:
        return '订单流程';
    }
  };

  return (
    <div style={containerStyle}>
      {/* 装饰性背景元素 */}
      <div style={{
        position: 'absolute',
        top: '-50%',
        right: '-50%',
        width: '100%',
        height: '100%',
        background: 'radial-gradient(circle, rgba(59, 130, 246, 0.05) 0%, transparent 70%)',
        pointerEvents: 'none'
      }} />
      
      <div style={titleStyle}>
        {getActionIcon()}
        {getActionTitle()}
      </div>
      
      <Steps
        className="order-status-flow"
        current={currentStepIndex}
        size="default"
        direction="horizontal"
        style={{
          position: 'relative',
          zIndex: 1,
          margin: '0 auto',
          width: '100%',
          maxWidth: '800px',
          padding: '16px 0'
        }}
        items={steps.map((step, index) => {
          const stepStatus = getStepStatus(step.status);
          const isActive = stepStatus === 'process';
          const isCompleted = stepStatus === 'finish';
          const isCancelled = step.title === '已取消';
          
          return {
            title: (
              <div style={{
                fontSize: '14px',
                fontWeight: isActive ? 600 : 500,
                color: isCancelled && isCompleted ? '#ef4444' : 
                       isCancelled && isActive ? '#ef4444' : 
                       isCompleted ? '#10b981' : 
                       isActive ? '#3b82f6' : '#64748b',
                transition: 'all 0.3s ease',
                textAlign: 'center',
                whiteSpace: 'nowrap',
                marginTop: '8px'
              }}>
                {step.title}
              </div>
            ),
            // 移除description以简化显示
            description: null,
            status: stepStatus,
            icon: (() => {
              const stepStatus = getStepStatus(step.status);
              const isCancelled = step.title === '已取消';
              
              return (
                <div 
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    width: '32px',
                    height: '32px',
                    borderRadius: '50%',
                    background: 'transparent',
                    border: 'none',
                    boxShadow: isCompleted 
                      ? '0 2px 8px rgba(16, 185, 129, 0.2), 0 0 0 2px rgba(16, 185, 129, 0.06)'
                      : isActive 
                      ? '0 2px 8px rgba(59, 130, 246, 0.2), 0 0 0 2px rgba(59, 130, 246, 0.06)'
                      : '0 1px 3px rgba(0, 0, 0, 0.04)',
                    transition: 'all 0.3s cubic-bezier(0.4, 0, 0.2, 1)',
                    transform: isActive ? 'scale(1.05)' : 'scale(1)',
                    animation: isActive ? 'pulse 2s infinite' : 'none',
                    position: 'relative',
                    overflow: 'hidden'
                  }}
                  data-cancelled={isCancelled ? 'true' : 'false'}
                >
                  {/* 内部光效 */}
                  {(isActive || isCompleted) && (
                    <div style={{
                      position: 'absolute',
                      top: '1px',
                      left: '1px',
                      right: '1px',
                      bottom: '1px',
                      borderRadius: '50%',
                      background: 'linear-gradient(45deg, rgba(255,255,255,0.3) 0%, transparent 50%, rgba(255,255,255,0.1) 100%)',
                      pointerEvents: 'none'
                    }} />
                  )}
                  
                  <style>
                    {`
                      @keyframes pulse {
                        0%, 100% { opacity: 1; }
                        50% { opacity: 0.85; }
                      }
                      @keyframes glow {
                        0% { box-shadow: 0 4px 16px rgba(59, 130, 246, 0.3), 0 0 0 3px rgba(59, 130, 246, 0.08); }
                        100% { box-shadow: 0 6px 20px rgba(59, 130, 246, 0.4), 0 0 0 4px rgba(59, 130, 246, 0.12); }
                      }
                      @keyframes bounce {
                        0%, 20%, 50%, 80%, 100% { transform: translateY(0); }
                        40% { transform: translateY(-3px); }
                        60% { transform: translateY(-2px); }
                      }
                    `}
                  </style>
                  
                  {(() => {
                    const getStepIcon = () => {
                      // 特殊处理：如果是最后一个步骤且当前状态是已完成的最终状态
                      if (index === steps.length - 1 && isFinalStatus(currentStatus) && currentStatus !== 'cancelled') {
                        // 最后一个节点直接显示完成状态
                        return <CheckOutlined style={{ 
                          color: '#ffffff', 
                          fontSize: '18px',
                          fontWeight: 'bold'
                        }} />;
                      }
                      
                      // 根据步骤类型和状态选择合适的图标
                      if (stepStatus === 'process') {
                        // 如果当前步骤是最后一个步骤，直接显示完成图标
                        const isLastStep = index === steps.length - 1;
                        
                        if (isLastStep) {
                          // 最后一个节点直接显示完成状态
                          return <CheckOutlined style={{ 
                            color: '#ffffff', 
                            fontSize: '18px',
                            fontWeight: 'bold'
                          }} />;
                        }
                        
                        if (step.title === '创建') {
                          return <RocketOutlined style={{ 
                            color: '#ffffff', 
                            fontSize: '18px',
                            fontWeight: 'bold'
                          }} />;
                        } else if (step.title === '处理中') {
                          return <SettingOutlined style={{ 
                            color: '#ffffff', 
                            fontSize: '18px',
                            animation: 'spin 2s linear infinite'
                          }} />;
                        } else if (step.title === '归还中') {
                          return <SyncOutlined style={{ 
                            color: '#ffffff', 
                            fontSize: '18px',
                            animation: 'spin 2s linear infinite'
                          }} />;
                        } else if (step.title === '已取消') {
                          return <CloseCircleOutlined style={{ 
                            color: '#ffffff', 
                            fontSize: '18px'
                          }} />;
                        } else {
                          return <LoadingOutlined style={{ 
                            color: '#ffffff', 
                            fontSize: '18px',
                            animation: 'spin 2s linear infinite'
                          }} />;
                        }
                      } else if (stepStatus === 'finish') {
                        if (step.title === '已取消') {
                          return <CloseCircleOutlined style={{ 
                            color: '#ffffff', 
                            fontSize: '18px'
                          }} />;
                        } else {
                          return <CheckOutlined style={{ 
                            color: '#ffffff', 
                            fontSize: '18px',
                            fontWeight: 'bold'
                          }} />;
                        }
                      } else {
                        if (step.title === '创建') {
                          return <RocketOutlined style={{ 
                            color: '#9ca3af', 
                            fontSize: '18px'
                          }} />;
                        } else if (step.title === '处理中') {
                          return <SettingOutlined style={{ 
                            color: '#9ca3af', 
                            fontSize: '18px'
                          }} />;
                        } else if (step.title === '归还中') {
                          return <SyncOutlined style={{ 
                            color: '#9ca3af', 
                            fontSize: '18px'
                          }} />;
                        } else if (step.title === '完成') {
                          return <CheckCircleOutlined style={{ 
                            color: '#9ca3af', 
                            fontSize: '18px'
                          }} />;
                        } else if (step.title === '已取消') {
                          return <CloseCircleOutlined style={{ 
                            color: '#9ca3af', 
                            fontSize: '18px'
                          }} />;
                        } else {
                          return <PauseCircleOutlined style={{ 
                            color: '#9ca3af', 
                            fontSize: '18px'
                          }} />;
                        }
                      }
                    };
                    
                    return getStepIcon();
                  })()}
                </div>
              );
            })()
          };
        })}
      />
      
      <style>
        {`
          @keyframes spin {
            from { transform: rotate(0deg); }
            to { transform: rotate(360deg); }
          }
          
          @keyframes pulse {
            0%, 100% { opacity: 1; transform: scale(1); }
            50% { opacity: 0.8; transform: scale(1.05); }
          }
          
          .ant-steps-item-process .ant-steps-item-icon,
          .ant-steps-item-finish .ant-steps-item-icon,
          .ant-steps-item-wait .ant-steps-item-icon {
            background: transparent !important;
            border: none !important;
            box-shadow: none !important;
            outline: none !important;
            width: 40px !important;
            height: 40px !important;
            display: flex !important;
            align-items: center !important;
            justify-content: center !important;
            border-radius: 50% !important;
            position: relative !important;
          }
          
          .ant-steps-item-icon {
            background: transparent !important;
            border: none !important;
            box-shadow: none !important;
            outline: none !important;
            width: 40px !important;
            height: 40px !important;
            display: flex !important;
            align-items: center !important;
            justify-content: center !important;
            border-radius: 50% !important;
            position: relative !important;
          }
          
          .ant-steps-item-process .ant-steps-item-icon {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%) !important;
            box-shadow: 0 4px 12px rgba(102, 126, 234, 0.3) !important;
            animation: pulse 2s ease-in-out infinite !important;
          }
          
          .ant-steps-item-finish .ant-steps-item-icon {
            background: linear-gradient(135deg, #10b981 0%, #059669 100%) !important;
            box-shadow: 0 4px 12px rgba(16, 185, 129, 0.3) !important;
          }
          
          .ant-steps-item-wait .ant-steps-item-icon {
            background: linear-gradient(135deg, #f1f5f9 0%, #e2e8f0 100%) !important;
            border: 2px solid #cbd5e1 !important;
          }
          
          .ant-steps-item-finish .ant-steps-item-icon[data-cancelled="true"] {
            background: linear-gradient(135deg, #ef4444 0%, #dc2626 100%) !important;
            box-shadow: 0 4px 12px rgba(239, 68, 68, 0.3) !important;
          }
          
          .ant-steps-item-process .ant-steps-item-icon[data-cancelled="true"] {
            background: linear-gradient(135deg, #ef4444 0%, #dc2626 100%) !important;
            box-shadow: 0 4px 12px rgba(239, 68, 68, 0.3) !important;
            animation: pulse 2s ease-in-out infinite !important;
          }
          
          .ant-steps-item {
            padding-left: 12px !important;
            padding-right: 12px !important;
            flex: 1 !important;
          }
          
          .ant-steps-item-tail {
            padding: 0 8px !important;
            top: 20px !important;
          }
          
          .ant-steps-item-tail::after {
            background: linear-gradient(90deg, #e2e8f0 0%, #cbd5e1 50%, #e2e8f0 100%) !important;
            height: 3px !important;
            border-radius: 2px !important;
            position: relative !important;
            top: 50% !important;
            transform: translateY(-50%) !important;
          }
          
          .ant-steps-item-finish .ant-steps-item-tail::after {
            background: linear-gradient(90deg, #10b981 0%, #059669 50%, #10b981 100%) !important;
            box-shadow: 0 2px 8px rgba(16, 185, 129, 0.2) !important;
          }
          
          .ant-steps-item-process .ant-steps-item-tail::after {
            background: linear-gradient(90deg, #667eea 0%, #764ba2 50%, #667eea 100%) !important;
            box-shadow: 0 2px 8px rgba(102, 126, 234, 0.2) !important;
          }
          
          .ant-steps-item-title {
            line-height: 1.4 !important;
            margin-top: 12px !important;
            font-size: 14px !important;
            font-weight: 500 !important;
            color: #374151 !important;
            text-align: center !important;
            letter-spacing: 0.5px !important;
          }
          
          .ant-steps-item-process .ant-steps-item-title {
            color: #667eea !important;
            font-weight: 600 !important;
          }
          
          .ant-steps-item-finish .ant-steps-item-title {
            color: #059669 !important;
            font-weight: 600 !important;
          }
          
          .ant-steps-item-wait .ant-steps-item-title {
            color: #9ca3af !important;
          }
          
          .ant-steps-item-description {
            margin-top: 4px !important;
            font-size: 12px !important;
            color: #6b7280 !important;
            text-align: center !important;
            line-height: 1.3 !important;
          }
          
          .ant-steps-item-process .ant-steps-item-description {
            color: #667eea !important;
          }
          
          .ant-steps-item-finish .ant-steps-item-description {
            color: #059669 !important;
          }
          
          .ant-steps {
            padding: 24px 16px !important;
            background: linear-gradient(135deg, #f8fafc 0%, #f1f5f9 100%) !important;
            border-radius: 16px !important;
            box-shadow: 0 4px 20px rgba(0, 0, 0, 0.05) !important;
            border: 1px solid rgba(255, 255, 255, 0.8) !important;
          }
        `}
      </style>
    </div>
  );
};

export default OrderStatusFlow;