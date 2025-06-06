import React, { useState, useEffect, useRef, useCallback } from 'react';
import { Table, Card, Input, Button, Space, Modal, Tabs, Tag, message } from 'antd';
import { SearchOutlined, NodeIndexOutlined, PartitionOutlined } from '@ant-design/icons';
import request from '../../utils/request';
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
  const [nodeDetail] = useState<NodeDetailInfo | null>(null); // setNodeDetail removed as it's unused
  const [nodeDetailVisible, setNodeDetailVisible] = useState<boolean>(false);
  const containerRef = useRef<HTMLDivElement | null>(null);
  const graphRef = useRef<any>(null);
  // fullscreen and setFullscreen removed as they are unused
  const [viewMode] = useState<'graph' | 'table'>('table'); // setViewMode removed as it's unused, 修改默认值为表格

  // 模拟获取数据
  useEffect(() => {
    const fetchData = async () => {
      setLoading(true);
      try {
        // 通过API获取数据
        const response = await request.get('/api/calico/nodes');
        setData(response.data);
        setLoading(false);
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
        style: { fill: '#e6f7ff', stroke: '#1890ff', lineWidth: 2 },
        size: 40,
        labelCfg: { position: 'bottom', style: { fill: '#000', fontSize: 12, fontWeight: 'bold' } },
      });

      // 查找所有连接到此 RR 的节点
      const connectedNodes = data.filter(d => d.rr === node.rr && d.id !== node.id); 
      if (!connectedNodes.find(cn => cn.id === node.id)) {
        connectedNodes.push(node);
      }

      connectedNodes.forEach((connNode) => {
        const nodeId = `node-${connNode.id}`; // 节点唯一 ID
        const nodeLabel = connNode.cluster; // 节点标签
        
        // 添加集群节点,增加样式
        nodes.push({ 
          id: nodeId, 
          label: nodeLabel, 
          type: 'node',
          originalData: connNode,
          style: { fill: '#f6ffed', stroke: '#52c41a', lineWidth: 2 },
          size: 30,
        });
        edges.push({ 
          id: `edge-${rrId}-${nodeId}`,
          source: rrId, 
          target: nodeId,
          style: { stroke: '#aaa', lineWidth: 1 },
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
            style: { fill: '#fffbe6', stroke: '#faad14', lineWidth: 1 },
            size: 20,
            labelCfg: { position: 'bottom', style: { fill: '#666', fontSize: 8 } },
          });
          edges.push({ 
            id: `edge-${nodeId}-${poolId}`,
            source: nodeId, 
            target: poolId,
            style: { stroke: '#ddd', lineWidth: 1 },
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
        style: { fill: '#e6f7ff', stroke: '#1890ff', lineWidth: 2 },
        size: 40,
        labelCfg: { position: 'bottom', style: { fill: '#000', fontSize: 12, fontWeight: 'bold' } },
      });

      const connectedNodes = data.filter(d => d.peer.startsWith(peerId) && d.id !== node.id);
      if (!connectedNodes.find(cn => cn.id === node.id)) {
        connectedNodes.push(node);
      }

      connectedNodes.forEach((connNode) => {
        const nodeId = `node-${connNode.id}`;
        
        // 添加集群节点,添加样式
        nodes.push({ 
          id: nodeId, 
          label: connNode.cluster, 
          type: 'node',
          originalData: connNode,
          style: { fill: '#f6ffed', stroke: '#52c41a', lineWidth: 2 },
          size: 30,
        });
        edges.push({ 
          id: `edge-${peerId}-${nodeId}`,
          source: peerId, 
          target: nodeId,
          style: { stroke: '#aaa', lineWidth: 1 },
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
            style: { fill: '#fffbe6', stroke: '#faad14', lineWidth: 1 },
            size: 20,
            labelCfg: { position: 'bottom', style: { fill: '#666', fontSize: 8 } },
          });
          edges.push({ 
            id: `edge-${nodeId}-${poolId}`,
            source: nodeId, 
            target: poolId,
            style: { stroke: '#ddd', lineWidth: 1 },
          });
        });
      });
    }
    
    console.log('Prepared topo data for G6 5.x with styles:', { nodes, edges });
    setTopoData({ nodes, edges });
    
    // 直接设置为拓扑视图可见，不再使用模态框的全屏状态
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
            { title: 'IP地址', dataIndex: 'ip', key: 'ip' },
            {
              title: '状态',
              dataIndex: 'status',
              key: 'status',
              render: (status: string) => (
                <Tag color={status === 'active' ? 'success' : 'error'}>{status}</Tag>
              ),
            },
            { title: 'Pod', dataIndex: 'pod', key: 'pod' },
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
              <div>CIDR: 10.244.0.0/16</div> {/* Example data */}
              <div>已分配IP: 24/256</div> {/* Example data */}
            </Card>
          ))}
        </div>
      ),
    },
  ];

  // --- Topology Rendering Logic (useCallback) ---

  // 静态拓扑图创建辅助函数 (useCallback)
  // NOTE: This function creates DOM elements directly. Ensure dependencies are correct.
  const createStaticTopology = useCallback((container: HTMLElement) => {
    // Clear previous content
    container.innerHTML = '';

    // 使用 HTML 元素创建一个动态悬浮的拓扑图
    const topo = document.createElement('div');
    topo.style.position = 'relative';
    topo.style.width = '400px'; // Example size, adjust as needed
    topo.style.height = '300px'; // Example size, adjust as needed
    
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
    const styleId = 'topo-animations';
    let style = document.getElementById(styleId);
    if (!style) {
        style = document.createElement('style');
        style.id = styleId;
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
        `;
        document.head.appendChild(style);
    }


    // --- Node/Edge Creation Logic ---
    const nodes = topoData.nodes;
    const edges = topoData.edges;
    const nodeElements: { [key: string]: HTMLElement } = {};
    const lines: { line: SVGLineElement, source: HTMLElement, target: HTMLElement }[] = [];

    // 创建中心节点
    const centerNodeData = nodes.find(n => n.type === 'rr');
    if (!centerNodeData) return;

    const centerNode = document.createElement('div');
    centerNode.id = centerNodeData.id;
    centerNode.textContent = centerNodeData.label.split('\n')[0];
    centerNode.title = centerNodeData.label;
    centerNode.style.position = 'absolute';
    centerNode.style.left = '180px';
    centerNode.style.top = '130px';
    centerNode.style.width = `${centerNodeData.size || 40}px`;
    centerNode.style.height = `${centerNodeData.size || 40}px`;
    centerNode.style.borderRadius = '50%';
    centerNode.style.backgroundColor = centerNodeData.style?.fill || '#e6f7ff';
    centerNode.style.border = `2px solid ${centerNodeData.style?.stroke || '#1890ff'}`;
    centerNode.style.display = 'flex';
    centerNode.style.alignItems = 'center';
    centerNode.style.justifyContent = 'center';
    centerNode.style.fontSize = '10px';
    centerNode.style.fontWeight = 'bold';
    centerNode.style.cursor = 'grab';
    centerNode.style.zIndex = '10';
    centerNode.style.boxShadow = '0 4px 12px rgba(0,0,0,0.1)';
    centerNode.style.animation = 'pulse 2s infinite ease-in-out';
    centerNode.dataset.tooltip = centerNodeData.label;

    const updateTooltipPosition = (e: MouseEvent) => {
        if (tooltip.style.display === 'block') {
          const topoRect = topo.getBoundingClientRect();
          const x = e.clientX - topoRect.left + 15;
          const y = e.clientY - topoRect.top + 15;
          tooltip.style.left = `${x}px`;
          tooltip.style.top = `${y}px`;
        }
      };

    centerNode.addEventListener('mouseover', (e) => {
      tooltip.textContent = centerNode.dataset.tooltip || '';
      tooltip.style.display = 'block';
      updateTooltipPosition(e);
    });
    centerNode.addEventListener('mousemove', updateTooltipPosition);
    centerNode.addEventListener('mouseout', () => {
      tooltip.style.display = 'none';
    });
    
    topo.appendChild(centerNode);
    nodeElements[centerNode.id] = centerNode;
    
    // 创建 SVG 画布用于连线
    const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
    svg.style.position = 'absolute';
    svg.style.width = '100%';
    svg.style.height = '100%';
    svg.style.top = '0';
    svg.style.left = '0';
    svg.style.pointerEvents = 'none';
    topo.insertBefore(svg, topo.firstChild);

    // 更新线条位置函数
    const updateLines = () => {
        lines.forEach(({ line, source, target }) => {
          const sourceRect = source.getBoundingClientRect();
          const targetRect = target.getBoundingClientRect();
          const topoRect = topo.getBoundingClientRect();
          const x1 = sourceRect.left - topoRect.left + sourceRect.width / 2;
          const y1 = sourceRect.top - topoRect.top + sourceRect.height / 2;
          const x2 = targetRect.left - topoRect.left + targetRect.width / 2;
          const y2 = targetRect.top - topoRect.top + targetRect.height / 2;
          line.setAttribute('x1', String(x1));
          line.setAttribute('y1', String(y1));
          line.setAttribute('x2', String(x2));
          line.setAttribute('y2', String(y2));
        });
      };

    // 创建 SVG 线条函数
    const createLine = (startNode: HTMLElement, endNode: HTMLElement, options: { color?: string, dashArray?: string } = {}) => {
        const line = document.createElementNS('http://www.w3.org/2000/svg', 'line');
        line.style.stroke = options.color || '#ccc';
        line.style.strokeWidth = '1.5';
        if (options.dashArray) {
          line.style.strokeDasharray = options.dashArray;
          line.style.animation = 'flow 1s linear infinite';
        }
        svg.appendChild(line);
        return { line, source: startNode, target: endNode };
      };

    // 拖动功能实现
    const makeDraggable = (element: HTMLElement) => {
        let isDragging = false;
        let offsetX = 0, offsetY = 0;
        let startX = 0, startY = 0;

        element.addEventListener('mousedown', (e) => {
          isDragging = true;
          startX = e.clientX;
          startY = e.clientY;
          offsetX = element.offsetLeft;
          offsetY = element.offsetTop;
          element.style.cursor = 'grabbing';
          element.style.zIndex = '100';
          e.preventDefault();

          const onMouseMove = (e: MouseEvent) => {
            if (!isDragging) return;
            const dx = e.clientX - startX;
            const dy = e.clientY - startY;
            const newLeft = offsetX + dx;
            const newTop = offsetY + dy;
            element.style.left = `${newLeft}px`;
            element.style.top = `${newTop}px`;
            updateLines();
            // If dragging a cluster node, move its blocks too (requires updateGroupElements logic)
          };

          const onMouseUp = () => {
            if (!isDragging) return;
            isDragging = false;
            element.style.cursor = 'grab';
            element.style.zIndex = element.id === centerNode.id ? '10' : '5';
            document.removeEventListener('mousemove', onMouseMove);
            document.removeEventListener('mouseup', onMouseUp);
          };

          document.addEventListener('mousemove', onMouseMove);
          document.addEventListener('mouseup', onMouseUp);
        });
      };

    makeDraggable(centerNode); // Make center node draggable

    // 创建其他节点和连线
    const connectedNodesData = nodes.filter(n => n.type !== 'rr');
    const angleStep = (2 * Math.PI) / (connectedNodesData.length || 1);
    const radius = 120;

    // Group nodes and blocks (simplified grouping logic)
    const nodeMap: { [key: string]: any } = {};
    const blockMap: { [key: string]: any } = {};
    connectedNodesData.forEach(n => {
        if (n.type === 'node') nodeMap[n.id] = n;
        else if (n.type === 'block') blockMap[n.id] = n;
    });

    Object.values(nodeMap).forEach((nodeData, index) => {
        const angle = index * angleStep;
        const nodeX = 180 + radius * Math.cos(angle);
        const nodeY = 130 + radius * Math.sin(angle);

        const node = document.createElement('div');
        node.id = nodeData.id;
        node.textContent = nodeData.label;
        node.title = nodeData.label;
        node.style.position = 'absolute';
        node.style.left = `${nodeX}px`;
        node.style.top = `${nodeY}px`;
        node.style.width = `${nodeData.size || 30}px`;
        node.style.height = `${nodeData.size || 30}px`;
        node.style.borderRadius = '50%';
        node.style.backgroundColor = nodeData.style?.fill || '#f6ffed';
        node.style.border = `2px solid ${nodeData.style?.stroke || '#52c41a'}`;
        node.style.display = 'flex';
        node.style.alignItems = 'center';
        node.style.justifyContent = 'center';
        node.style.fontSize = '9px';
        node.style.cursor = 'pointer'; // Changed to pointer, draggable optional
        node.style.zIndex = '5';
        node.style.animation = 'float 3s infinite ease-in-out';
        node.dataset.tooltip = nodeData.label;

        node.addEventListener('mouseover', (e) => { /* ... tooltip logic ... */ });
        node.addEventListener('mousemove', updateTooltipPosition);
        node.addEventListener('mouseout', () => { /* ... tooltip logic ... */ });

        topo.appendChild(node);
        nodeElements[node.id] = node;
        // makeDraggable(node); // Optionally make cluster nodes draggable

        // Connect center node and cluster node
        const centerEdge = edges.find(e => (e.source === centerNode.id && e.target === node.id) || (e.source === node.id && e.target === centerNode.id));
        if (centerEdge) {
            lines.push(createLine(centerNode, node, { color: centerEdge.style?.stroke || '#aaa' }));
        }


        // Find and create Block nodes connected to this cluster node
        const blockRadius = 40;
        let blockIndex = 0;
        edges.forEach(edge => {
            let blockData: any = null;
            if (edge.source === node.id && blockMap[edge.target]) {
                blockData = blockMap[edge.target];
            } else if (edge.target === node.id && blockMap[edge.source]) {
                // Handle potential reverse edge definition if needed
                // blockData = blockMap[edge.source];
            }

            if (blockData) {
                const blockAngleStepLocal = (2 * Math.PI) / (edges.filter(e => e.source === node.id && blockMap[e.target]).length || 1); // Adjust angle based on actual blocks for this node
                const blockAngle = blockIndex * blockAngleStepLocal;
                const blockX = nodeX + blockRadius * Math.cos(blockAngle);
                const blockY = nodeY + blockRadius * Math.sin(blockAngle);

                const blockNode = document.createElement('div');
                blockNode.id = blockData.id;
                blockNode.textContent = blockData.label.length > 10 ? blockData.label.substring(0, 7) + '...' : blockData.label;
                blockNode.title = blockData.label;
                blockNode.style.position = 'absolute';
                blockNode.style.left = `${blockX}px`;
                blockNode.style.top = `${blockY}px`;
                blockNode.style.width = `${blockData.size || 20}px`;
                blockNode.style.height = `${blockData.size || 20}px`;
                blockNode.style.borderRadius = '4px';
                blockNode.style.backgroundColor = blockData.style?.fill || '#fffbe6';
                blockNode.style.border = `1px solid ${blockData.style?.stroke || '#faad14'}`;
                blockNode.style.display = 'flex';
                blockNode.style.alignItems = 'center';
                blockNode.style.justifyContent = 'center';
                blockNode.style.fontSize = '8px';
                blockNode.style.color = '#666';
                blockNode.style.zIndex = '3';
                blockNode.dataset.tooltip = blockData.label;

                blockNode.addEventListener('mouseover', (e) => { /* ... tooltip logic ... */ });
                blockNode.addEventListener('mousemove', updateTooltipPosition);
                blockNode.addEventListener('mouseout', () => { /* ... tooltip logic ... */ });

                topo.appendChild(blockNode);
                nodeElements[blockNode.id] = blockNode;

                // Connect cluster node and Block node
                lines.push(createLine(node, blockNode, { color: edge.style?.stroke || '#ddd', dashArray: '3,3' }));
                blockIndex++;
            }
        });
    });


    updateLines(); // Initial line positioning

    container.appendChild(topo);

    // Return a cleanup function for the style element
    return () => {
      if (style && style.parentNode) {
        style.parentNode.removeChild(style);
      }
    };

  }, []); // Dependencies for createStaticTopology

  // 将G6图初始化逻辑抽取为单独的函数 (useCallback)
  const initGraph = useCallback(() => {
    if (!containerRef.current || !currentNode || !topoVisible) {
      console.log('无法初始化拓扑图: 容器不存在或数据不完整');
      return undefined; // Return undefined if not initialized
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
      canvas.style.height = 'calc(100% - 40px)'; // Adjust height if legend takes space
      canvas.style.display = 'flex';
      canvas.style.alignItems = 'center';
      canvas.style.justifyContent = 'center';
      containerRef.current.appendChild(canvas);


      // 创建静态拓扑图并获取清理函数
      const cleanupStaticTopology = createStaticTopology(canvas); // Call the memoized function

      // 返回清理函数
      return cleanupStaticTopology;
    }
    return undefined; // Return undefined if containerRef.current is null
  }, [currentNode, topoVisible, createStaticTopology]); // Dependencies for initGraph

  // 初始化拓扑图 - 当模态框打开时
  useEffect(() => {
    console.log('Topo useEffect triggered:', { topoVisible, hasContainer: !!containerRef.current, currentNode, viewMode });

    let cleanupGraph: (() => void) | undefined | null = null;
    let timerId: NodeJS.Timeout | null = null;

    if (topoVisible && currentNode && viewMode === 'graph') {
      // 延迟一点初始化，确保容器元素已经完全渲染
      timerId = setTimeout(() => {
         cleanupGraph = initGraph(); // Call memoized initGraph and store cleanup
      }, 300);
    } else {
      // Clean up existing graph instance if viewMode changes or topo becomes invisible
      if (graphRef.current) {
        graphRef.current.destroy();
        graphRef.current = null;
      }
       // Also cleanup static topology if necessary
       if (containerRef.current) {
           containerRef.current.innerHTML = ''; // Simple cleanup for static HTML
       }
    }

    // Cleanup function for the effect
    return () => {
      if (timerId) clearTimeout(timerId); // Clear timeout if component unmounts or deps change
      if (graphRef.current) {
        graphRef.current.destroy();
        graphRef.current = null;
      }
      if (typeof cleanupGraph === 'function') {
          cleanupGraph(); // Call cleanup returned by initGraph
      }
       // Also cleanup static topology if necessary on unmount
       const container = containerRef.current;
       if (container) {
           // Check if cleanup is needed or just clear innerHTML
           // container.innerHTML = '';
       }
    };
  // Dependencies: topoVisible, currentNode, viewMode, initGraph (stable due to useCallback)
  }, [topoVisible, currentNode, viewMode, initGraph]);


  // 下载路由信息 (示例函数，需要实现具体逻辑)
  const downloadRoute = (node: CalicoNode, type: 'RR' | 'AnchorLeaf' | 'ALL' | 'Detail') => {
    message.info(`准备下载 ${node.cluster} 的 ${type} 路由信息...`);
    // TODO: 实现下载逻辑, e.g., call API endpoint
  };

  // 表格列定义
  const columns: ColumnsType<CalicoNode> = [
    { title: '集群', dataIndex: 'cluster', key: 'cluster', width: 150, fixed: 'left' },
    { title: 'AS号', dataIndex: 'as', key: 'as', width: 80 },
    {
      title: 'Peer',
      dataIndex: 'peer',
      key: 'peer',
      width: 250,
      render: (text: string) => {
        const parts = text.split(' : ');
        return (
          <>
            <Tag color="blue">{parts[0]}</Tag>
            {parts[1] && <Tag color="cyan">{parts[1]}</Tag>}
          </>
        );
      },
    },
    { title: '节点类型', dataIndex: 'nodeType', key: 'nodeType', width: 150 },
    { title: 'POD类型', dataIndex: 'podType', key: 'podType', width: 120 },
    {
      title: 'IPv4 Pool',
      dataIndex: 'poolV4',
      key: 'poolV4',
      width: 300,
      render: (pools: string[]) => (
        <div style={{ maxHeight: '60px', overflowY: 'auto' }}>
            {pools.map((pool, index) => (
            <Tag key={index} color="geekblue" style={{ marginBottom: '4px' }}>{pool}</Tag>
            ))}
          </div>
      ),
    },
    {
      title: 'IPv6 Pool',
      dataIndex: 'poolV6',
      key: 'poolV6',
      width: 300,
      render: (pools: string[]) => (
        <div style={{ maxHeight: '60px', overflowY: 'auto' }}>
            {pools.map((pool, index) => (
            <Tag key={index} color="purple" style={{ marginBottom: '4px' }}>{pool}</Tag>
            ))}
          </div>
      ),
    },
    { title: 'RR', dataIndex: 'rr', key: 'rr', width: 150 },
    {
      title: '操作',
      key: 'action',
      fixed: 'right',
      width: 300,
      render: (_: any, record: CalicoNode) => (
        <Space size={8} style={{ display: 'flex', flexWrap: 'wrap', gap: '8px' }}>
          <Button 
            size="small" 
            type="primary"
            ghost
            icon={<PartitionOutlined />}
            onClick={() => showTopo(record, 'rr')}
            disabled={!record.rr} // Disable if RR is empty
          >
            RR拓扑
          </Button>
          <Button 
            size="small" 
            type="primary"
            ghost
            icon={<NodeIndexOutlined />}
            onClick={() => showTopo(record, 'anchorleaf')}
            disabled={!record.peer || !record.peer.includes(':')} // Disable if peer format is wrong
          >
            AnchorLeaf拓扑
          </Button>
          <Button size="small" onClick={() => downloadRoute(record, 'RR')}>下载RR路由</Button>
          <Button size="small" onClick={() => downloadRoute(record, 'AnchorLeaf')}>下载AnchorLeaf路由</Button>
          <Button size="small" onClick={() => downloadRoute(record, 'ALL')}>下载ALL路由</Button>
          {/* <Button size="small" onClick={() => downloadRoute(record, 'Detail')}>下载Detail路由</Button> */}
        </Space>
      ),
    },
  ];

  // 过滤数据
  const filteredData = data.filter(item =>
    item.cluster.toLowerCase().includes(searchText.toLowerCase()) ||
    item.peer.toLowerCase().includes(searchText.toLowerCase()) ||
    item.rr.toLowerCase().includes(searchText.toLowerCase()) ||
    item.nodeType.toLowerCase().includes(searchText.toLowerCase())
  );

  return (
    <div style={{ padding: '24px' }}>
      <Card
        title="Calico 网络拓扑与路由信息"
        bordered={false}
        extra={
          <div style={{ display: 'flex', alignItems: 'center' }}>
            <Input
              placeholder="搜索集群、Peer、RR、节点类型..."
              prefix={<SearchOutlined />}
              value={searchText}
              onChange={(e) => setSearchText(e.target.value)}
              style={{ width: 300, marginRight: 16 }}
              allowClear
            />
            {/* 可以添加其他操作按钮 */}
          </div>
        }
      >
        <Table
          columns={columns}
          dataSource={filteredData}
          loading={loading}
          rowKey="id"
          scroll={{ x: 1800 }} // 调整滚动宽度
          pagination={{ pageSize: 10 }}
          size="small"
        />
      </Card>

      {/* 拓扑图展示区域 (不再使用 Modal) */}
      {topoVisible && (
        <div 
          style={{ 
            position: 'fixed',
            top: 0,
            left: 0,
            width: '100vw',
            height: '100vh',
            backgroundColor: 'rgba(0, 0, 0, 0.7)',
            zIndex: 1000, // Ensure it's above other content
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
            padding: '20px',
            boxSizing: 'border-box',
          }}
        >
          <Card
            title={`拓扑图: ${currentNode?.cluster} (${topoType === 'rr' ? 'RR' : 'AnchorLeaf'})`}
            style={{ width: '90%', height: '90%', display: 'flex', flexDirection: 'column' }}
            bodyStyle={{ flex: 1, overflow: 'hidden', padding: '16px' }} // Allow body to grow and hide overflow
            extra={
            <Space>
              <Button 
                  danger
                onClick={() => {
                  setTopoVisible(false);
                    // setFullscreen(false); // fullscreen state removed
                }}
              >
                  关闭
              </Button>
            </Space>
            }
          >
            {/* Conditionally render Node Details or Topology */}
            {viewMode === 'table' && nodeDetailVisible ? (
                 <Tabs defaultActiveKey="1" items={nodeDetailItems} />
            ) : (
                 <div ref={containerRef} style={{ width: '100%', height: '100%', border: '1px solid #eee', position: 'relative' }}>
                   {/* Static topology will be rendered here by initGraph */}
            </div>
            )}
          </Card>
        </div>
      )}

      {/* 节点详情 Modal (如果需要单独展示) */}
      <Modal
        title="节点详情"
        open={nodeDetailVisible && viewMode === 'table'} // Only show if table mode and detail requested
        onCancel={() => setNodeDetailVisible(false)}
        footer={null} // No footer needed if just displaying info
        width={800}
      >
        <Tabs defaultActiveKey="1" items={nodeDetailItems} />
      </Modal>
    </div>
  );
};

export default CalicoNetworkTopology;