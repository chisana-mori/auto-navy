import React, { useState, useEffect, useRef } from 'react';
import { Table, Card, Input, Button, Space, Modal, Tabs, Tag, message } from 'antd';
import { SearchOutlined, CloudServerOutlined, NodeIndexOutlined, PartitionOutlined, TableOutlined } from '@ant-design/icons';
import type { TabsProps } from 'antd';
import type { ColumnsType } from 'antd/es/table';
// 暂时注释掉 G6，使用纯 HTML 实现
// import { Graph } from '@antv/g6';

// 定义数据类型
interface CalicoNode {
  id: string;
  cluster: string;
  as: string;
  peer: string;
  nodeType: string;
  podType: string;
  poolV4: string[];
  poolV6: string[];
  rr: string;
}

interface IpInfo {
  ip: string;
  status: string;
  pod?: string;
}

interface NodeDetailInfo {
  ips: IpInfo[];
  pools: string[];
}

const CalicoNetworkTopology: React.FC = () => {
  const [loading, setLoading] = useState<boolean>(false);
  const [data, setData] = useState<CalicoNode[]>([]);
  const [searchText, setSearchText] = useState<string>('');
  const [topoVisible, setTopoVisible] = useState<boolean>(false);
  const [topoType, setTopoType] = useState<'rr' | 'anchorleaf'>('rr');
  const [currentNode, setCurrentNode] = useState<CalicoNode | null>(null);
  const [topoData, setTopoData] = useState<{ nodes: any[], edges: any[] }>({ nodes: [], edges: [] });
  const [nodeDetail, setNodeDetail] = useState<NodeDetailInfo | null>(null);
  const [nodeDetailVisible, setNodeDetailVisible] = useState<boolean>(false);
  const containerRef = useRef<HTMLDivElement | null>(null);
  const graphRef = useRef<any>(null);
  const [fullscreen, setFullscreen] = useState<boolean>(false);
  const [viewMode, setViewMode] = useState<'graph' | 'table'>('table'); // 修改默认值为表格

  // 模拟获取数据
  useEffect(() => {
    const fetchData = async () => {
      setLoading(true);
      try {
        // 实际项目中应该通过API获取数据
        // const response = await request.get('/api/calico/nodes');
        // setData(response.data);
        
        // --- 生成更复杂的模拟数据 ---
        const mockNodes: CalicoNode[] = [];
        const rrName = 'feature.node/rr';
        const anchorLeafPeerPrefix = '61201-feature.node/rr';
        const anchorLeafPeerIP = '29.23.254.165';
        const totalNodes = 20;
        const poolsPerNode = 10;
        const rrNodesCount = 10;

        for (let i = 1; i <= totalNodes; i++) {
          const isRRNode = i <= rrNodesCount;
          const clusterName = `cluster-${String(i).padStart(2, '0')}`;
          const nodeAS = '65001';
          
          const pools: string[] = [];
          for (let p = 1; p <= poolsPerNode; p++) {
            pools.push(`pool-${clusterName}-${p}`);
          }

          mockNodes.push({
            id: String(i),
            cluster: clusterName,
            as: nodeAS,
            // 前 10 个节点连接到 AnchorLeaf, 后 10 个连接到 RR
            peer: isRRNode ? `${anchorLeafPeerPrefix} : ${anchorLeafPeerIP}` : `some-other-peer-${i} : 10.0.0.${i}`, 
            nodeType: isRRNode ? 'K8S主机模式' : '测试K8S集群主机模式',
            podType: isRRNode ? 'K8SBASE' : 'K8S非业务APP',
            poolV4: pools,
            poolV6: [], // 简化，只用 IPv4 pools
            rr: isRRNode ? '' : rrName, // 前 10 个 rr 为空, 后 10 个指定 rr
          });
        }
        
        // 这里使用模拟数据
        setTimeout(() => {
          setData(mockNodes);
          setLoading(false);
        }, 500);
      } catch (error) {
        console.error('获取数据失败:', error);
        message.error('获取数据失败');
        setLoading(false);
      }
    };

    fetchData();
  }, []);

  // 显示表格详情
  const showTopo = (node: CalicoNode, type: 'rr' | 'anchorleaf') => {
    setTopoType(type);
    setCurrentNode(node);
    
    // --- 准备拓扑数据 --- 
    const nodes: any[] = [];
    const edges: any[] = [];

    if (type === 'rr') {
      const rrId = node.rr || `rr-${node.id}`; // RR 的唯一 ID
      const rrLabel = node.rr || `RR (${node.cluster})`;
      
      // 添加 RR 节点,添加类型和样式
      nodes.push({ 
        id: rrId, 
        label: rrLabel, 
        type: 'rr',
        style: {
          fill: '#e6f7ff',
          stroke: '#1890ff',
          lineWidth: 2,
        },
        size: 40,
        labelCfg: {
          position: 'bottom',
          style: {
            fill: '#000',
            fontSize: 12,
            fontWeight: 'bold',
          },
        },
      });

      // 查找所有连接到此 RR 的节点
      const connectedNodes = data.filter(d => d.rr === node.rr && d.id !== node.id); 
      if (!connectedNodes.find(cn => cn.id === node.id)) {
        connectedNodes.push(node);
      }

      connectedNodes.forEach((connNode, index) => {
        const nodeId = `node-${connNode.id}`; // 节点唯一 ID
        const nodeLabel = connNode.cluster; // 节点标签
        
        // 添加集群节点,增加样式
        nodes.push({ 
          id: nodeId, 
          label: nodeLabel, 
          type: 'node',
          originalData: connNode,
          style: {
            fill: '#f6ffed',
            stroke: '#52c41a',
            lineWidth: 2,
          },
          size: 30,
        });
        edges.push({ 
          id: `edge-${rrId}-${nodeId}`,
          source: rrId, 
          target: nodeId,
          style: {
            stroke: '#aaa',
            lineWidth: 1,
          },
        });

        // 添加节点的 IP Pool
        const pools = [...connNode.poolV4, ...connNode.poolV6];
        
        pools.forEach((pool, poolIndex) => {
          const poolId = `pool-${connNode.id}-${poolIndex}`; // IP Pool 唯一 ID
          
          // 添加 IP Pool 节点,增加样式
          nodes.push({ 
            id: poolId, 
            label: pool, 
            type: 'block',
            style: {
              fill: '#fffbe6',
              stroke: '#faad14',
              lineWidth: 1,
            },
            size: 20,
            labelCfg: {
              position: 'bottom',
              style: {
                fill: '#666',
                fontSize: 8,
              },
            },
          });
          edges.push({ 
            id: `edge-${nodeId}-${poolId}`,
            source: nodeId, 
            target: poolId,
            style: {
              stroke: '#ddd',
              lineWidth: 1,
            },
          });
        });
      });
    } else if (type === 'anchorleaf') {
      // AnchorLeaf 逻辑
      const peerId = node.peer.split(' : ')[0] || `peer-${node.id}`;
      const peerIp = node.peer.split(' : ')[1] || '';
      const peerLabel = `${peerId}${peerIp ? '\n' + peerIp : ''}`;
      
      // 添加中心节点,添加样式
      nodes.push({ 
        id: peerId, 
        label: peerLabel, 
        type: 'rr', // 保持用 rr type 表示中心
        style: {
          fill: '#e6f7ff',
          stroke: '#1890ff',
          lineWidth: 2,
        },
        size: 40,
        labelCfg: {
          position: 'bottom',
          style: {
            fill: '#000',
            fontSize: 12,
            fontWeight: 'bold',
          },
        },
      });

      const connectedNodes = data.filter(d => d.peer.startsWith(peerId) && d.id !== node.id);
      if (!connectedNodes.find(cn => cn.id === node.id)) {
        connectedNodes.push(node);
      }

      connectedNodes.forEach((connNode, index) => {
        const nodeId = `node-${connNode.id}`;
        
        // 添加集群节点,添加样式
        nodes.push({ 
          id: nodeId, 
          label: connNode.cluster, 
          type: 'node',
          originalData: connNode,
          style: {
            fill: '#f6ffed',
            stroke: '#52c41a',
            lineWidth: 2,
          },
          size: 30,
        });
        edges.push({ 
          id: `edge-${peerId}-${nodeId}`,
          source: peerId, 
          target: nodeId,
          style: {
            stroke: '#aaa',
            lineWidth: 1,
          },
        });

        // 添加节点的 IP Pool
        const pools = [...connNode.poolV4, ...connNode.poolV6];
        pools.forEach((pool, poolIndex) => {
          const poolId = `pool-${connNode.id}-${poolIndex}`;
          
          // 添加 IP Pool 节点,添加样式
          nodes.push({ 
            id: poolId, 
            label: pool, 
            type: 'block',
            style: {
              fill: '#fffbe6',
              stroke: '#faad14',
              lineWidth: 1,
            },
            size: 20,
            labelCfg: {
              position: 'bottom',
              style: {
                fill: '#666',
                fontSize: 8,
              },
            },
          });
          edges.push({ 
            id: `edge-${nodeId}-${poolId}`,
            source: nodeId, 
            target: poolId,
            style: {
              stroke: '#ddd',
              lineWidth: 1,
            },
          });
        });
      });
    }
    
    console.log('Prepared topo data for G6 5.x with styles:', { nodes, edges });
    setTopoData({ nodes, edges });
    
    // 直接设置为全屏模式，不再使用模态框
    setFullscreen(true);
    setTopoVisible(true);
  };

  // 节点详情标签页
  const nodeDetailItems: TabsProps['items'] = [
    {
      key: '1',
      label: 'IP分配',
      children: (
        <Table 
          dataSource={nodeDetail?.ips || []}
          rowKey="ip"
          pagination={false}
          columns={[
            {
              title: 'IP地址',
              dataIndex: 'ip',
              key: 'ip',
            },
            {
              title: '状态',
              dataIndex: 'status',
              key: 'status',
              render: (status: string) => (
                <Tag color={status === 'active' ? 'success' : 'error'}>
                  {status}
                </Tag>
              )
            },
            {
              title: 'Pod',
              dataIndex: 'pod',
              key: 'pod',
            }
          ]}
        />
      ),
    },
    {
      key: '2',
      label: 'Pool 信息',
      children: (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
          {nodeDetail?.pools.map((pool, index) => (
            <Card key={index} size="small" title={pool} style={{ marginBottom: '8px' }}>
              <div>CIDR: 10.244.0.0/16</div>
              <div>已分配IP: 24/256</div>
            </Card>
          ))}
        </div>
      ),
    },
  ];

  // 初始化拓扑图 - 当模态框打开时
  useEffect(() => {
    console.log('Topo useEffect triggered:', { topoVisible, hasContainer: !!containerRef.current, currentNode, topoData });
    
    if (topoVisible && currentNode && viewMode === 'graph') {
      // 延迟一点初始化，确保容器元素已经完全渲染
      setTimeout(() => initGraph(), 300);
    } else {
      if (graphRef.current) {
        graphRef.current.destroy();
        graphRef.current = null;
      }
    }

    return () => {
      if (graphRef.current) {
        graphRef.current.destroy();
        graphRef.current = null;
      }
    };
  }, [topoVisible, currentNode, viewMode]); // 添加viewMode依赖

  // 将G6图初始化逻辑抽取为单独的函数
  const initGraph = () => {
    if (!containerRef.current || !currentNode || !topoVisible) {
      console.log('无法初始化拓扑图: 容器不存在或数据不完整');
      return;
    }

    // 测量容器尺寸
    const width = containerRef.current.clientWidth || 800;
    const height = containerRef.current.clientHeight || 500;
    console.log('容器尺寸:', { width, height });

    // 清理旧内容
    if (containerRef.current) {
      containerRef.current.innerHTML = '';
    }

    // 不使用 G6，直接用 HTML 和 CSS 创建一个简单的网络拓扑图示例
    if (containerRef.current) {
      containerRef.current.style.position = 'relative';
      containerRef.current.style.overflow = 'hidden';
      
      // 创建简单的图例
      const legend = document.createElement('div');
      legend.style.position = 'absolute';
      legend.style.top = '10px';
      legend.style.right = '10px';
      legend.style.background = 'rgba(255,255,255,0.9)';
      legend.style.padding = '12px';
      legend.style.border = '1px solid #eee';
      legend.style.borderRadius = '4px';
      legend.style.boxShadow = '0 2px 8px rgba(0,0,0,0.1)';
      legend.style.zIndex = '10';
      legend.innerHTML = `
        <div style="margin-bottom: 10px; font-weight: bold; font-size: 14px; color: #333; border-bottom: 1px solid #eee; padding-bottom: 5px;">图例</div>
        <div style="display: flex; align-items: center; margin-bottom: 8px;">
          <div style="width: 18px; height: 18px; border-radius: 50%; background-color: #e6f7ff; border: 2px solid #1890ff; margin-right: 10px;"></div>
          <span>${topoType === 'rr' ? 'RR' : 'AnchorLeaf'} 节点</span>
        </div>
        <div style="display: flex; align-items: center; margin-bottom: 8px;">
          <div style="width: 16px; height: 16px; border-radius: 50%; background-color: #f6ffed; border: 2px solid #52c41a; margin-right: 10px;"></div>
          <span>集群节点</span>
        </div>
        <div style="display: flex; align-items: center;">
          <div style="width: 14px; height: 14px; border-radius: 50%; background-color: #fffbe6; border: 1px solid #faad14; margin-right: 10px;"></div>
          <span>Block</span>
        </div>
      `;
      containerRef.current.appendChild(legend);
      
      // 创建画布区域
      const canvas = document.createElement('div');
      canvas.style.width = '100%';
      canvas.style.height = '80%';
      canvas.style.display = 'flex';
      canvas.style.alignItems = 'center';
      canvas.style.justifyContent = 'center';
      containerRef.current.appendChild(canvas);
      
      // 创建静态拓扑图
      createStaticTopology(canvas);
    }
  };
  
  // 静态拓扑图创建辅助函数
  const createStaticTopology = (container: HTMLElement) => {
    // 使用 HTML 元素创建一个动态悬浮的拓扑图
    const topo = document.createElement('div');
    topo.style.position = 'relative';
    topo.style.width = '400px';
    topo.style.height = '300px';
    
    // 创建背景辅助网格（可选）
    const grid = document.createElement('div');
    grid.style.position = 'absolute';
    grid.style.width = '100%';
    grid.style.height = '100%';
    grid.style.background = 'radial-gradient(circle, rgba(0,0,0,0.02) 1px, transparent 1px)';
    grid.style.backgroundSize = '20px 20px';
    grid.style.opacity = '0.6';
    grid.style.pointerEvents = 'none';
    topo.appendChild(grid);
    
    // 创建提示框元素
    const tooltip = document.createElement('div');
    tooltip.style.position = 'absolute';
    tooltip.style.padding = '6px 10px';
    tooltip.style.background = 'rgba(0, 0, 0, 0.75)';
    tooltip.style.color = '#fff';
    tooltip.style.borderRadius = '4px';
    tooltip.style.fontSize = '12px';
    tooltip.style.pointerEvents = 'none';
    tooltip.style.zIndex = '100';
    tooltip.style.display = 'none';
    tooltip.style.maxWidth = '200px';
    tooltip.style.boxShadow = '0 2px 8px rgba(0, 0, 0, 0.15)';
    topo.appendChild(tooltip);
    
    // 添加悬浮动画样式
    const style = document.createElement('style');
    style.textContent = `
      @keyframes float {
        0% { transform: translateY(0px); }
        50% { transform: translateY(-5px); }
        100% { transform: translateY(0px); }
      }
      
      @keyframes pulse {
        0% { box-shadow: 0 0 5px rgba(24, 144, 255, 0.3); }
        50% { box-shadow: 0 0 15px rgba(24, 144, 255, 0.5); }
        100% { box-shadow: 0 0 5px rgba(24, 144, 255, 0.3); }
      }
      
      @keyframes flow {
        0% { stroke-dashoffset: 24; }
        100% { stroke-dashoffset: 0; }
      }
      
      .node-float {
        animation: float 3s ease-in-out infinite, pulse 2s ease-in-out infinite;
      }
      
      .node-float-slow {
        animation: float 4s ease-in-out infinite, pulse 3s ease-in-out infinite;
      }
      
      .flow-line {
        stroke-dasharray: 4, 2;
        animation: flow 1s linear infinite;
      }
    `;
    document.head.appendChild(style);
    
    // 中心节点 (RR 或 AnchorLeaf)
    const centerNode = document.createElement('div');
    centerNode.style.position = 'absolute';
    centerNode.style.width = '40px';
    centerNode.style.height = '40px';
    centerNode.style.borderRadius = '50%';
    centerNode.style.background = '#e6f7ff';
    centerNode.style.border = '2px solid #1890ff';
    centerNode.style.boxShadow = '0 0 10px rgba(24, 144, 255, 0.3)';
    centerNode.style.left = '180px';
    centerNode.style.top = '130px';
    centerNode.style.zIndex = '2';
    centerNode.style.cursor = 'pointer';
    centerNode.style.transition = 'transform 0.3s ease';
    centerNode.classList.add('node-float');
    topo.appendChild(centerNode);
    
    // 为中心节点添加鼠标悬停事件
    centerNode.addEventListener('mouseover', (e) => {
      tooltip.innerHTML = `${topoType === 'rr' ? 'RR节点' : 'AnchorLeaf节点'}`;
      tooltip.style.display = 'block';
      updateTooltipPosition(e);
      centerNode.style.transform = 'scale(1.1)';
    });
    
    centerNode.addEventListener('mousemove', updateTooltipPosition);
    
    centerNode.addEventListener('mouseout', () => {
      tooltip.style.display = 'none';
      centerNode.style.transform = 'scale(1)';
    });
    
    // 添加拖拽功能
    makeDraggable(centerNode);
    
    // 创建SVG容器用于绘制流动线条
    const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
    svg.style.position = 'absolute';
    svg.style.top = '0';
    svg.style.left = '0';
    svg.style.width = '100%';
    svg.style.height = '100%';
    svg.style.pointerEvents = 'none';
    svg.style.zIndex = '1';
    topo.appendChild(svg);
    
    // 节点位置
    const positions = [
      { left: '80px', top: '50px' },   // 左上
      { left: '280px', top: '50px' },  // 右上
      { left: '80px', top: '200px' },  // 左下
      { left: '280px', top: '200px' }, // 右下
    ];
    
    // 存储所有线条和节点的引用，用于更新连线位置
    const lines: { line: SVGLineElement, source: HTMLElement, target: HTMLElement }[] = [];
    
    // 存储节点和Block的分组，用于同步更新位置
    interface NodeBlockGroup {
      node: HTMLElement;
      blockNode: HTMLElement | null;
      groupElement: HTMLElement;
    }
    
    const groupElements: NodeBlockGroup[] = [];
    
    // 创建4个集群节点
    for (let i = 0; i < 4; i++) {
      // 集群节点
      const node = document.createElement('div');
      node.style.position = 'absolute';
      node.style.width = '30px';
      node.style.height = '30px';
      node.style.borderRadius = '50%';
      node.style.background = '#f6ffed';
      node.style.border = '2px solid #52c41a';
      node.style.boxShadow = '0 0 8px rgba(82, 196, 26, 0.2)';
      node.style.left = positions[i].left;
      node.style.top = positions[i].top;
      node.style.zIndex = '2';
      node.style.cursor = 'move'; // 改为move光标
      node.style.transition = 'transform 0.3s ease';
      node.classList.add('node-float-slow');
      // 错开动画
      node.style.animationDelay = `${i * 0.5}s`;
      topo.appendChild(node);
      
      // 为集群节点添加悬停事件
      node.addEventListener('mouseover', (e) => {
        tooltip.innerHTML = `节点${i+1}<br/>集群: 集群${i+1}<br/>AS: 65001`;
        tooltip.style.display = 'block';
        updateTooltipPosition(e);
        node.style.transform = 'scale(1.1)';
      });
      
      node.addEventListener('mousemove', updateTooltipPosition);
      
      node.addEventListener('mouseout', () => {
        tooltip.style.display = 'none';
        node.style.transform = 'scale(1)';
      });
      
      node.addEventListener('click', () => message.info(`点击了集群节点 ${i+1}`));
      
      // 添加拖拽功能
      makeDraggable(node);
      
      // 计算连线初始位置
      const dx = parseInt(positions[i].left) - 180;
      const dy = parseInt(positions[i].top) - 130;
      
      // 节点到中心的连线 - 使用SVG绘制
      createLine(centerNode, node);
      
      // 存储线条关联
      // lines.push({ line, source: centerNode, target: node });
      
      // 创建虚线圆圈框，包含节点和其Block
      const nodeGroup = document.createElement('div');
      nodeGroup.style.position = 'absolute';
      nodeGroup.style.width = '70px';
      nodeGroup.style.height = '70px';
      nodeGroup.style.borderRadius = '50%';
      nodeGroup.style.border = '2px dashed #91d5ff';
      nodeGroup.style.left = `${parseInt(positions[i].left) - 20}px`;
      nodeGroup.style.top = `${parseInt(positions[i].top) - 20}px`;
      nodeGroup.style.zIndex = '0'; // 放在节点和连线的下层
      topo.appendChild(nodeGroup);
      
      // 为两个节点存储关联的分组框
      const nodeBlockGroup: NodeBlockGroup = { node, blockNode: null, groupElement: nodeGroup };
      
      // Block节点
      const blockNode = document.createElement('div');
      const blockDx = dx > 0 ? 20 : -20;
      const blockDy = dy > 0 ? 20 : -20;
      
      blockNode.style.position = 'absolute';
      blockNode.style.width = '20px';
      blockNode.style.height = '20px';
      blockNode.style.borderRadius = '50%';
      blockNode.style.background = '#fffbe6';
      blockNode.style.border = '1px solid #faad14';
      blockNode.style.boxShadow = '0 0 5px rgba(250, 173, 20, 0.2)';
      blockNode.style.left = `${parseInt(positions[i].left) + blockDx}px`;
      blockNode.style.top = `${parseInt(positions[i].top) + blockDy}px`;
      blockNode.style.zIndex = '2';
      blockNode.style.cursor = 'default'; // 不允许拖拽，改为默认光标
      blockNode.style.transition = 'transform 0.3s ease';
      blockNode.classList.add('node-float-slow');
      // 错开动画
      blockNode.style.animationDelay = `${i * 0.5 + 0.25}s`;
      topo.appendChild(blockNode);
      
      // 为Block节点添加悬停事件
      blockNode.addEventListener('mouseover', (e) => {
        tooltip.innerHTML = `Block ${i+1}<br/>CIDR: 10.244.${i}.0/24<br/>所属IP Pool: pool-cluster-${i+1}<br/>类型: 主机级Block<br/>已分配: 18个地址`;
        tooltip.style.display = 'block';
        updateTooltipPosition(e);
        blockNode.style.transform = 'scale(1.1)';
      });
      
      blockNode.addEventListener('mousemove', updateTooltipPosition);
      
      blockNode.addEventListener('mouseout', () => {
        tooltip.style.display = 'none';
        blockNode.style.transform = 'scale(1)';
      });
      
      blockNode.addEventListener('click', () => message.info(`点击了Block ${i+1}`));
      
      // Block节点不可拖拽，注释掉这行
      // makeDraggable(blockNode);
      
      // 将Block节点添加到nodeBlockGroup中
      nodeBlockGroup.blockNode = blockNode;
      
      // 添加到groupElements数组中，用于拖拽时更新
      groupElements.push(nodeBlockGroup);
      
      // 集群节点到Block的连线 - 使用SVG
      createLine(node, blockNode, { color: '#faad14', dashArray: '3,2' });
      
      // 存储线条关联
      // lines.push({ line: blockLine, source: node, target: blockNode });
    }
    
    // 添加SVG渐变定义
    const defs = document.createElementNS('http://www.w3.org/2000/svg', 'defs');
    const gradient = document.createElementNS('http://www.w3.org/2000/svg', 'linearGradient');
    gradient.setAttribute('id', 'gradient');
    gradient.setAttribute('x1', '0%');
    gradient.setAttribute('y1', '0%');
    gradient.setAttribute('x2', '100%');
    gradient.setAttribute('y2', '0%');
    
    const stop1 = document.createElementNS('http://www.w3.org/2000/svg', 'stop');
    stop1.setAttribute('offset', '0%');
    stop1.setAttribute('stop-color', '#1890ff');
    
    const stop2 = document.createElementNS('http://www.w3.org/2000/svg', 'stop');
    stop2.setAttribute('offset', '100%');
    stop2.setAttribute('stop-color', '#52c41a');
    
    gradient.appendChild(stop1);
    gradient.appendChild(stop2);
    defs.appendChild(gradient);
    svg.appendChild(defs);
    
    container.appendChild(topo);
    
    // 更新tooltip位置的函数
    function updateTooltipPosition(e: MouseEvent) {
      const x = e.clientX;
      const y = e.clientY;
      const rect = topo.getBoundingClientRect();
      const topoX = x - rect.left;
      const topoY = y - rect.top;
      
      // 防止tooltip超出容器边界
      tooltip.style.left = `${topoX + 15}px`;
      tooltip.style.top = `${topoY + 15}px`;
    }
    
    // 添加拖拽功能的辅助函数
    function makeDraggable(element: HTMLElement) {
      let isDragging = false;
      let startX = 0;
      let startY = 0;
      let startLeft = 0;
      let startTop = 0;
      
      // 鼠标按下时记录初始位置
      element.addEventListener('mousedown', (e) => {
        e.preventDefault(); // 防止文本选择
        isDragging = true;
        
        // 记录初始鼠标位置
        startX = e.clientX;
        startY = e.clientY;
        
        // 记录元素初始位置（移除px单位并转为数字）
        startLeft = parseInt(element.style.left) || 0;
        startTop = parseInt(element.style.top) || 0;
        
        element.style.zIndex = '10'; // 提高正在拖拽的元素的层级
        
        // 停止浮动动画
        element.style.animation = 'none';
        
        // 拖拽时隐藏tooltip
        tooltip.style.display = 'none';
        
        // 鼠标移动时更新位置
        const onMouseMove = (e: MouseEvent) => {
          if (!isDragging) return;
          
          // 计算鼠标移动的距离
          const deltaX = e.clientX - startX;
          const deltaY = e.clientY - startY;
          
          // 根据初始位置和鼠标移动距离计算新位置
          const newLeft = startLeft + deltaX;
          const newTop = startTop + deltaY;
          
          // 更新元素位置
          element.style.left = `${newLeft}px`;
          element.style.top = `${newTop}px`;
          
          // 更新与此元素相关的所有线条
          updateLines();
          
          // 更新与此元素相关的分组框
          updateGroupElements(element, newLeft, newTop);
        };
        
        // 鼠标释放时结束拖拽
        const onMouseUp = () => {
          if (isDragging) {
            element.style.zIndex = '2';
            isDragging = false;
            
            // 恢复浮动动画
            element.style.animation = '';
            
            // 移除临时事件监听器
            document.removeEventListener('mousemove', onMouseMove);
            document.removeEventListener('mouseup', onMouseUp);
            document.removeEventListener('mouseleave', onMouseUp);
          }
        };
        
        // 添加临时事件监听器
        document.addEventListener('mousemove', onMouseMove);
        document.addEventListener('mouseup', onMouseUp);
        document.addEventListener('mouseleave', onMouseUp);
      });
    }
    
    // 更新线条位置和角度的函数
    function updateLines() {
      lines.forEach(({ line, source, target }) => {
        // 获取源节点和目标节点的位置和尺寸
        const sourceLeft = parseInt(source.style.left) || 0;
        const sourceTop = parseInt(source.style.top) || 0;
        const sourceWidth = parseInt(source.style.width) || 0;
        const sourceHeight = parseInt(source.style.height) || 0;
        
        const targetLeft = parseInt(target.style.left) || 0;
        const targetTop = parseInt(target.style.top) || 0;
        const targetWidth = parseInt(target.style.width) || 0;
        const targetHeight = parseInt(target.style.height) || 0;
        
        // 计算节点中心点
        const sourceX = sourceLeft + sourceWidth / 2;
        const sourceY = sourceTop + sourceHeight / 2;
        const targetX = targetLeft + targetWidth / 2;
        const targetY = targetTop + targetHeight / 2;
        
        // 更新SVG线条的位置
        line.setAttribute('x1', sourceX.toString());
        line.setAttribute('y1', sourceY.toString());
        line.setAttribute('x2', targetX.toString());
        line.setAttribute('y2', targetY.toString());
      });
    }
    
    // 创建线条的辅助函数
    function createLine(startNode: HTMLElement, endNode: HTMLElement, options: { color?: string, dashArray?: string } = {}) {
      // 初始创建时只根据节点的style属性获取位置
      const startLeft = parseInt(startNode.style.left) || 0;
      const startTop = parseInt(startNode.style.top) || 0;
      const endLeft = parseInt(endNode.style.left) || 0;
      const endTop = parseInt(endNode.style.top) || 0;
      
      // 计算节点中心点
      const startWidth = parseInt(startNode.style.width) || 0;
      const startHeight = parseInt(startNode.style.height) || 0;
      const endWidth = parseInt(endNode.style.width) || 0;
      const endHeight = parseInt(endNode.style.height) || 0;
      
      const startX = startLeft + startWidth / 2;
      const startY = startTop + startHeight / 2;
      const endX = endLeft + endWidth / 2;
      const endY = endTop + endHeight / 2;
      
      // 创建SVG线条
      const line = document.createElementNS('http://www.w3.org/2000/svg', 'line');
      
      // 设置线条起点和终点
      line.setAttribute('x1', startX.toString());
      line.setAttribute('y1', startY.toString());
      line.setAttribute('x2', endX.toString());
      line.setAttribute('y2', endY.toString());
      
      // 设置线条样式
      line.setAttribute('stroke', options.color || 'url(#gradient)');
      line.setAttribute('stroke-width', '2');
      line.setAttribute('stroke-dasharray', options.dashArray || '4,2');
      line.classList.add('flow-line');
      
      svg.appendChild(line);
      
      // 存储线条关联
      lines.push({ line, source: startNode, target: endNode });
      
      return line;
    }
    
    // 更新分组元素位置的函数
    function updateGroupElements(element: HTMLElement, newLeft: number, newTop: number) {
      // 查找与当前拖拽元素相关的分组
      for (const group of groupElements) {
        if (group.node === element) {
          // 如果拖拽的是集群节点，更新整个分组的位置
          group.groupElement.style.left = `${newLeft - 20}px`;
          group.groupElement.style.top = `${newTop - 20}px`;
          
          // 如果Block节点存在，也需要同步移动，保持相对位置不变
          if (group.blockNode) {
            // 保持Block与节点的相对位置，固定偏移量
            const dx = parseInt(group.node.style.left) > 180 ? 20 : -20;
            const dy = parseInt(group.node.style.top) > 130 ? 20 : -20;
            
            // 更新Block位置，跟随节点移动
            group.blockNode.style.left = `${newLeft + dx}px`;
            group.blockNode.style.top = `${newTop + dy}px`;
          }
        }
        // Block节点不再可拖拽，这里不需要处理Block被拖拽的情况
      }
    }
    
    // 定期更新线条动画（可选，如果想要更复杂的动画效果）
    const animationTimer = setInterval(() => {
      lines.forEach(({ line }) => {
        // 重新应用动画类以保持动画流畅
        line.classList.remove('flow-line');
        // 触发重排的替代方法
        setTimeout(() => {
          line.classList.add('flow-line');
        }, 10);
      });
    }, 5000);
    
    // 返回清理函数，在组件卸载时调用
    return () => {
      clearInterval(animationTimer);
    };
  };

  // 下载路由
  const downloadRoute = (node: CalicoNode, type: 'RR' | 'AnchorLeaf' | 'ALL' | 'Detail') => {
    message.success(`开始下载 ${node.cluster} 的 ${type} 路由信息`);
    // 实际项目中应该调用API下载路由信息
  };

  // 定义表格列
  const columns: ColumnsType<CalicoNode> = [
    {
      title: '集群',
      dataIndex: 'cluster',
      key: 'cluster',
      width: 180,
      filteredValue: searchText ? [searchText] : null,
      onFilter: (value: boolean | React.Key, record: CalicoNode) => {
        const filterValue = String(value).toLowerCase();
        return record.cluster.toLowerCase().includes(filterValue);
      },
    },
    {
      title: 'AS',
      dataIndex: 'as',
      key: 'as',
      width: 80,
    },
    {
      title: '集群peer',
      dataIndex: 'peer',
      key: 'peer',
      width: 220,
      render: (text: string) => {
        if (!text) return '-';
        const parts = text.split(' : ');
        return (
          <>
            <div>{parts[0]}</div>
            {parts[1] && <div>{parts[1]}</div>}
          </>
        );
      },
    },
    {
      title: 'node类',
      dataIndex: 'nodeType',
      key: 'nodeType',
      width: 180,
    },
    {
      title: 'pod类',
      dataIndex: 'podType',
      key: 'podType',
      width: 130,
    },
    {
      title: 'pool-v4',
      dataIndex: 'poolV4',
      key: 'poolV4',
      width: 180,
      render: (pools: string[]) => {
        if (!pools || pools.length === 0) return '-';
        return (
          <div style={{ maxHeight: '80px', overflowY: 'auto' }}>
            {pools.map((pool, index) => (
              <div key={index}>
                <span>{pool}</span>
              </div>
            ))}
          </div>
        );
      },
    },
    {
      title: 'pool-v6',
      dataIndex: 'poolV6',
      key: 'poolV6',
      width: 180,
      render: (pools: string[]) => {
        if (!pools || pools.length === 0) return '-';
        return (
          <div>
            {pools.map((pool, index) => (
              <div key={index}>
                <span>{pool}</span>
              </div>
            ))}
          </div>
        );
      },
    },
    {
      title: 'rr',
      dataIndex: 'rr',
      key: 'rr',
      width: 180,
    },
    {
      title: '路由下载',
      key: 'action',
      fixed: 'right' as const,
      width: 180,
      render: (_: any, record: CalicoNode) => (
        <Space size={8} style={{ display: 'flex', flexWrap: 'wrap', gap: '8px' }}>
          <Button 
            size="small" 
            type="link"
            onClick={() => showTopo(record, 'rr')}
          >
            查看RR详情
          </Button>
          <Button 
            size="small" 
            type="link"
            onClick={() => showTopo(record, 'anchorleaf')}
          >
            查看AnchorLeaf详情
          </Button>
          <Button 
            size="small" 
            type="link"
            onClick={() => downloadRoute(record, 'ALL')}
          >
            ALL
          </Button>
          <Button 
            size="small" 
            type="link"
            onClick={() => downloadRoute(record, 'Detail')}
          >
            Detail
          </Button>
          <Button 
            size="small"
            type="link"
            onClick={() => downloadRoute(record, 'Detail')}
          >
            历史追踪
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: '20px' }}>
      <Card
        title={
          <div style={{ display: 'flex', alignItems: 'center' }}>
            <NodeIndexOutlined style={{ fontSize: '20px', marginRight: '8px' }} />
            <span>Calico网络管理</span>
          </div>
        }
        extra={
          <Space>
            <Input
              placeholder="输入Calico集群..."
              value={searchText}
              onChange={(e) => setSearchText(e.target.value)}
              allowClear
              style={{ width: 200 }}
              prefix={<SearchOutlined />}
            />
            <Button type="primary">查询</Button>
          </Space>
        }
      >
        <Table
          columns={columns}
          dataSource={data}
          rowKey="id"
          loading={loading}
          bordered
          size="middle"
          scroll={{ x: 'max-content' }}
          pagination={{
            defaultPageSize: 10,
            showQuickJumper: true,
            showSizeChanger: true,
            showTotal: (total) => `共 ${total} 条`,
          }}
        />
      </Card>

      {/* 网络拓扑图全屏模式 */}
      {topoVisible && (
        <div 
          style={{ 
            position: 'fixed',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            zIndex: 1000,
            background: '#fff',
            padding: '16px',
            display: 'flex',
            flexDirection: 'column',
          }}
        >
          <div style={{ 
            display: 'flex', 
            justifyContent: 'space-between', 
            alignItems: 'center',
            marginBottom: '16px'
          }}>
            <div style={{ display: 'flex', alignItems: 'center' }}>
              <PartitionOutlined style={{ marginRight: '8px' }} />
              <span style={{ fontSize: '18px', fontWeight: 'bold' }}>
                {topoType === 'rr' ? 'RR' : 'AnchorLeaf'} 集群详情 - {currentNode?.cluster}
              </span>
            </div>
            <Space>
              <Button 
                type="primary" 
                onClick={() => {
                  setTopoVisible(false);
                  setFullscreen(false); // 关闭时重置全屏状态
                }}
              >
                返回
              </Button>
            </Space>
          </div>
          <div 
            ref={containerRef} 
            style={{ 
              flex: 1,
              border: '1px solid #f0f0f0', 
              borderRadius: '2px',
              position: 'relative',
              display: 'none', // 永久隐藏拓扑图容器
            }}
          />

          {/* 表格视图 */}
          <div style={{ flex: 1, overflow: 'auto', display: 'flex' }}>
            {/* 左侧节点列表 */}
            <div style={{ width: '40%', borderRight: '1px solid #f0f0f0', padding: '0 16px 16px 0' }}>
              <div style={{ fontWeight: 'bold', marginBottom: '16px', fontSize: '16px' }}>
                当前集群信息
              </div>
              <div style={{ 
                padding: '15px', 
                border: '1px solid #f0f0f0', 
                borderRadius: '4px', 
                marginBottom: '16px', 
                background: '#fafafa'
              }}>
                <div style={{ marginBottom: '8px' }}>
                  <span style={{ fontWeight: 'bold', marginRight: '8px' }}>集群名称:</span>
                  <span>{currentNode?.cluster}</span>
                </div>
                <div style={{ marginBottom: '8px' }}>
                  <span style={{ fontWeight: 'bold', marginRight: '8px' }}>AS号:</span>
                  <span>{currentNode?.as}</span>
                </div>
                <div style={{ marginBottom: '8px' }}>
                  <span style={{ fontWeight: 'bold', marginRight: '8px' }}>节点类型:</span>
                  <span>{currentNode?.nodeType}</span>
                </div>
                <div>
                  <span style={{ fontWeight: 'bold', marginRight: '8px' }}>Pod类型:</span>
                  <span>{currentNode?.podType}</span>
                </div>
              </div>
              
              <div style={{ fontWeight: 'bold', marginBottom: '16px', fontSize: '16px' }}>
                连接信息
              </div>
              <div style={{ 
                padding: '15px', 
                border: '1px solid #f0f0f0', 
                borderRadius: '4px',
                background: '#fafafa'
              }}>
                <div style={{ marginBottom: '8px' }}>
                  <span style={{ fontWeight: 'bold', marginRight: '8px' }}>Peer:</span>
                  <span>{
                    currentNode?.peer ? currentNode.peer.split(' : ').join(' - ') : '-'
                  }</span>
                </div>
                <div>
                  <span style={{ fontWeight: 'bold', marginRight: '8px' }}>RR:</span>
                  <span>{currentNode?.rr || '-'}</span>
                </div>
              </div>
            </div>
            
            {/* 右侧Block信息 */}
            <div style={{ flex: 1, padding: '0 0 16px 16px' }}>
              <div style={{ fontWeight: 'bold', marginBottom: '16px', fontSize: '16px' }}>
                {`Block信息 - ${currentNode?.cluster}`}
              </div>
              <Table
                columns={[
                  {
                    title: 'Block名称',
                    dataIndex: 'label',
                    key: 'label',
                  },
                  {
                    title: 'CIDR',
                    key: 'cidr',
                    render: (_, record, index) => `10.244.${index}.0/24`
                  },
                  {
                    title: '状态',
                    key: 'status',
                    render: () => <Tag color="success">正常</Tag>
                  },
                  {
                    title: '已分配IP',
                    key: 'allocatedIps',
                    render: (_, record, index) => `${12 + index}/254`
                  }
                ]}
                dataSource={
                  // 只显示当前集群相关的Block
                  currentNode?.poolV4.map((pool, index) => ({
                    id: `pool-${currentNode.id}-${index}`,
                    label: pool,
                    key: `pool-${currentNode.id}-${index}`,
                  })) || []
                }
                bordered
                size="middle"
                pagination={false}
                locale={{ emptyText: '该集群没有关联的Block' }}
              />
            </div>
          </div>
        </div>
      )}

      {/* 节点详情模态框 */}
      <Modal
        title={
          <div style={{ display: 'flex', alignItems: 'center' }}>
            <CloudServerOutlined style={{ marginRight: '8px' }} />
            节点详情
          </div>
        }
        open={nodeDetailVisible}
        onCancel={() => setNodeDetailVisible(false)}
        width={700}
        footer={null}
        destroyOnClose={true}
      >
        <Tabs items={nodeDetailItems} />
      </Modal>
    </div>
  );
};

export default CalicoNetworkTopology;