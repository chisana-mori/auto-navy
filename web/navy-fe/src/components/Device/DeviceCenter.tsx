import React, { useState, useEffect, useCallback, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import { Tabs, Card, Table, Button, message, Space, Tooltip, Modal, Form, Input, Tag, Typography, Select, Spin, Pagination } from 'antd';
// import type { SelectProps } from 'antd/es/select';
import {
  DatabaseOutlined,
  ReloadOutlined,
  EditOutlined,
  PlayCircleOutlined,
  DeleteOutlined,
  FileSearchOutlined,
  PlusOutlined,
  SyncOutlined
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { Device, DeviceListResponse } from '../../types/device';
import { FilterGroup } from '../../types/deviceQuery';
import { getDeviceList, updateDeviceGroup } from '../../services/deviceService';
import { getDeviceFeatureDetails } from '../../services/deviceQueryService';
import { queryDevices, getQueryTemplates, getQueryTemplate, getDeviceFieldValues, deleteQueryTemplate } from '../../services/deviceQueryService';
import { generateQuerySummary } from '../../utils/queryUtils';
import SimpleQueryPanel from './SimpleQueryPanel';
import QuerySummary from './QuerySummary';
import AdvancedQueryPanel from './AdvancedQueryPanel';
import '../../styles/device-center.css';

const { TabPane } = Tabs;
const { Paragraph } = Typography;
const { Option } = Select;

// 查询结果类型
type QueryResult = {
  devices: Device[];
  pagination: {
    current: number;
    pageSize: number;
    total: number;
  };
  loading: boolean;
  lastUpdated: Date | null;
};

// 查询状态类型
type QueryState = {
  mode: 'simple' | 'advanced' | 'template';
  simpleParams: {
    keyword: string;
    results: QueryResult; // 基本查询的独立结果
  };
  advancedParams: {
    groups: FilterGroup[];
    sourceTemplateId?: number;  // 模板来源ID
    sourceTemplateName?: string;  // 模板来源名称
    results: QueryResult; // 高级查询的独立结果
  };
  templateParams: {
    templateId: number | null;
    templateName: string;
    results: QueryResult; // 模板查询的独立结果
  };
};

// 初始结果状态
const initialQueryResult: QueryResult = {
  devices: [],
  pagination: {
    current: 1,
    pageSize: 10,
    total: 0,
  },
  loading: false,
  lastUpdated: null,
};

// 初始状态
const initialQueryState: QueryState = {
  mode: 'simple',
  simpleParams: {
    keyword: '',
    results: { ...initialQueryResult },
  },
  advancedParams: {
    groups: [],
    results: { ...initialQueryResult },
  },
  templateParams: {
    templateId: null,
    templateName: '',
    results: { ...initialQueryResult },
  },
};

const DeviceCenter: React.FC = () => {
  const navigate = useNavigate();
  const location = window.location;

  // 在组件初始化时立即从URL获取查询参数
  const getInitialQueryParams = () => {
    const searchParams = new URLSearchParams(location.search);
    return {
      tab: searchParams.get('tab') as 'simple' | 'advanced' | 'template' | null,
      templateId: searchParams.get('templateId') ? parseInt(searchParams.get('templateId')!) : null,
    };
  };

  // 获取初始参数
  const initialParams = getInitialQueryParams();

  // 根据URL参数设置初始状态
  const getInitialState = (): QueryState => {
    // 如果URL中有tab参数，使用它作为初始模式
    const initialMode = initialParams.tab || 'simple';

    return {
      ...initialQueryState,
      mode: initialMode,
      // 如果是高级查询模式且有模板ID，设置加载状态为true
      advancedParams: {
        ...initialQueryState.advancedParams,
        results: {
          ...initialQueryState.advancedParams.results,
          loading: initialParams.tab === 'advanced' && initialParams.templateId !== null
        }
      }
    };
  };

  const [queryState, setQueryState] = useState<QueryState>(getInitialState());
  const [roleEditVisible, setRoleEditVisible] = useState(false);
  const [editingDevice, setEditingDevice] = useState<Device | null>(null);
  const [groupOptions, setGroupOptions] = useState<string[]>([]);
  const [loadingGroupOptions, setLoadingGroupOptions] = useState(false);
  const [templates, setTemplates] = useState<any[]>([]);
  const [templateSearchKeyword, setTemplateSearchKeyword] = useState('');
  const [templatePagination, setTemplatePagination] = useState({
    current: 1,
    pageSize: 8,
    total: 0,
  });
  const [roleForm] = Form.useForm();
  const [initialLoadComplete, setInitialLoadComplete] = useState(false);

  // 防重复请求的引用
  const isExecutingQuery = useRef(false);

  // 加载模板列表
  // eslint-disable-next-line react-hooks/exhaustive-deps
  const loadTemplates = useCallback(async () => {
    try {
      const { current, pageSize } = templatePagination;

      const templatesResponse = await getQueryTemplates({
        page: current,
        size: pageSize
      });

      // 更新模板列表和分页信息
      setTemplates(templatesResponse.list);
      setTemplatePagination(prev => ({
        ...prev,
        total: templatesResponse.total,
        // 如果返回的页码和每页数量与当前不同，更新它们
        current: templatesResponse.page || prev.current,
        pageSize: templatesResponse.size || prev.pageSize
      }));
    } catch (error) {
      console.error('加载模板列表失败:', error);
      message.error('加载模板列表失败');
    }
  }, [templatePagination.current, templatePagination.pageSize]); // 只依赖必要的分页参数

  // 处理模板搜索
  const handleTemplateSearch = (value: string) => {
    setTemplateSearchKeyword(value);
    // 重置到第一页
    setTemplatePagination(prev => ({
      ...prev,
      current: 1,
    }));
  };

  // 处理模板分页变化
  const handleTemplatePageChange = (page: number, pageSize?: number) => {
    // 更新分页状态
    setTemplatePagination(prev => ({
      ...prev,
      current: page,
      pageSize: pageSize || prev.pageSize,
    }));
  };

  // 当模板分页状态改变时加载数据
  useEffect(() => {
    // 如果尚未完成初始加载，不执行模板重新加载
    if (!initialLoadComplete) {
      return;
    }
    
    loadTemplates();
  }, [loadTemplates, initialLoadComplete]);

  // 加载模板列表和初始设备数据
  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => {
    // 如果已经完成初始加载，不再重复执行
    if (initialLoadComplete) {
      return;
    }

    const loadInitialData = async () => {
      try {
        // 获取URL参数 - 使用已经在组件初始化时获取的参数
        const { tab, templateId } = initialParams;

        // 直接调用模板API而不是依赖loadTemplates，避免循环依赖
        const templatesPromise = (async () => {
          try {
            const templatesResponse = await getQueryTemplates({
              page: 1,
              size: 8
            });
            setTemplates(templatesResponse.list);
            setTemplatePagination(prev => ({
              ...prev,
              total: templatesResponse.total,
              current: templatesResponse.page || prev.current,
              pageSize: templatesResponse.size || prev.pageSize
            }));
          } catch (error) {
            console.error('加载模板列表失败:', error);
            message.error('加载模板列表失败');
          }
        })();

        // 如果URL中有templateId参数且是高级查询模式，优先加载模板并执行查询
        if (templateId && (tab === 'advanced' || tab === null)) {
          try {
            console.log('从URL参数加载模板:', templateId);

            // 加载模板
            const template = await getQueryTemplate(templateId);
            console.log('模板加载成功:', template);

            if (template && template.groups) {
              // 确保每个筛选块的 key 和 field 字段同步
              const processedGroups = template.groups.map(group => ({
                ...group,
                blocks: group.blocks.map(block => {
                  // 创建一个新的块对象，避免修改原始对象
                  const processedBlock = { ...block };

                  // 确保 key 和 field 字段同步
                  if (processedBlock.field && !processedBlock.key) {
                    processedBlock.key = processedBlock.field;
                  } else if (processedBlock.key && !processedBlock.field) {
                    processedBlock.field = processedBlock.key;
                  }

                  return processedBlock;
                })
              }));

              console.log('处理后的查询组:', processedGroups);

              // 立即执行查询，不等待状态更新，并防止重复执行
              if (!isExecutingQuery.current) {
                console.log('直接执行模板查询');
                isExecutingQuery.current = true;
                
                try {
                  const queryResponse = await queryDevices({
                    groups: processedGroups,
                    page: 1,
                    size: 10,
                  });

                  console.log('模板查询响应:', queryResponse);

                  // 一次性更新所有状态，减少重渲染
                  if (queryResponse) {
                    setQueryState(prev => ({
                      ...prev,
                      mode: 'advanced',
                      advancedParams: {
                        ...prev.advancedParams,
                        groups: processedGroups,
                        sourceTemplateId: templateId,
                        sourceTemplateName: template.name,
                        results: {
                          devices: queryResponse.list || [],
                          pagination: {
                            current: 1,
                            pageSize: 10,
                            total: queryResponse.total || 0,
                          },
                          loading: false,
                          lastUpdated: new Date(),
                        }
                      }
                    }));

                    // 等待DOM更新后显示成功消息
                    setTimeout(() => {
                      message.success('查询成功');
                    }, 100);

                    // 标记初始加载完成
                    setInitialLoadComplete(true);

                    // 等待模板列表加载完成
                    await templatesPromise;
                    return;
                  }
                } finally {
                  isExecutingQuery.current = false;
                }
              }
            }
          } catch (error) {
            console.error('模板加载或查询失败:', error);
            message.error('加载模板失败');
            isExecutingQuery.current = false;

            // 重置加载状态
            setQueryState(prev => ({
              ...prev,
              advancedParams: {
                ...prev.advancedParams,
                results: {
                  ...prev.advancedParams.results,
                  loading: false,
                }
              }
            }));
          }
        }

        // 如果没有模板ID或模板加载失败，加载初始设备列表
        console.log('加载初始设备列表');
        setQueryState(prev => ({
          ...prev,
          simpleParams: {
            ...prev.simpleParams,
            results: {
              ...prev.simpleParams.results,
              loading: true,
            }
          }
        }));

        const response = await getDeviceList({
          page: 1,
          size: 10,
          keyword: '',
        });

        if (response) {
          setQueryState(prev => ({
            ...prev,
            simpleParams: {
              ...prev.simpleParams,
              results: {
                devices: response.list || [],
                pagination: {
                  current: 1,
                  pageSize: 10,
                  total: response.total || 0,
                },
                loading: false,
                lastUpdated: new Date(),
              }
            }
          }));
        }

        // 等待模板列表加载完成
        await templatesPromise;

        // 标记初始加载完成
        setInitialLoadComplete(true);
      } catch (error) {
        console.error('初始数据加载失败:', error);
        setQueryState(prev => ({
          ...prev,
          simpleParams: {
            ...prev.simpleParams,
            results: {
              ...prev.simpleParams.results,
              loading: false,
            }
          }
        }));

        // 即使出错也标记初始加载完成
        setInitialLoadComplete(true);
      }
    };

    loadInitialData();
  }, [initialParams]);

  // 处理多行查询关键字
  const processMultilineKeyword = (keyword: string): string => {
    // 如果关键字中包含换行符，则将其分割为多个关键字
    if (keyword.includes('\n')) {
      // 分割成多行，过滤空行，并用空格连接
      const lines = keyword.split('\n')
        .map(line => line.trim())
        .filter(line => line.length > 0);

      if (lines.length > 1) {
        // 多行查询时，使用 OR 连接多个关键字
        return lines.join(' OR ');
      } else if (lines.length === 1) {
        // 只有一行非空内容
        return lines[0];
      }
    }

    // 如果没有换行符或所有行都是空的，返回原始关键字
    return keyword;
  };

  // 执行查询的通用函数 - 改为普通函数避免依赖项问题
  const executeQuery = async (mode?: 'simple' | 'advanced' | 'template') => {
    // 防止重复请求
    if (isExecutingQuery.current) {
      console.log('查询正在执行中，跳过重复请求');
      return;
    }

    const currentMode = mode || queryState.mode;
    const { simpleParams, advancedParams, templateParams } = queryState;

    // 获取当前标签页的分页信息
    let currentPagination;
    switch (currentMode) {
      case 'simple':
        currentPagination = simpleParams.results.pagination;
        break;
      case 'advanced':
        currentPagination = advancedParams.results.pagination;
        break;
      case 'template':
        currentPagination = templateParams.results.pagination;
        break;
      default:
        currentPagination = simpleParams.results.pagination;
        break;
    }

    try {
      isExecutingQuery.current = true;

      // 设置加载状态
      setQueryState(prev => {
        const newState = { ...prev };
        switch (currentMode) {
          case 'simple':
            newState.simpleParams = {
              ...prev.simpleParams,
              results: {
                ...prev.simpleParams.results,
                loading: true,
              }
            };
            break;
          case 'advanced':
            newState.advancedParams = {
              ...prev.advancedParams,
              results: {
                ...prev.advancedParams.results,
                loading: true,
              }
            };
            break;
          case 'template':
            newState.templateParams = {
              ...prev.templateParams,
              results: {
                ...prev.templateParams.results,
                loading: true,
              }
            };
            break;
        }
        return newState;
      });

      let response: DeviceListResponse | undefined;

      switch (currentMode) {
        case 'simple':
          // 处理多行查询
          const processedKeyword = processMultilineKeyword(simpleParams.keyword);
          response = await getDeviceList({
            page: currentPagination.current,
            size: currentPagination.pageSize,
            keyword: processedKeyword,
          });
          break;
        case 'advanced':
          response = await queryDevices({
            groups: advancedParams.groups,
            page: currentPagination.current,
            size: currentPagination.pageSize,
          });
          break;
        case 'template':
          console.log('Template case in executeQuery, templateParams:', templateParams);
          if (templateParams.templateId) {
            console.log('Template ID exists, fetching template:', templateParams.templateId);
            const template = await getQueryTemplate(templateParams.templateId);
            console.log('Template fetched:', template);
            response = await queryDevices({
              groups: template.groups,
              page: currentPagination.current,
              size: currentPagination.pageSize,
            });
            console.log('Query response:', response);
          } else {
            console.log('No template ID found, skipping query');
          }
          break;
      }

      if (response) {
        // 使用类型断言来确保 TypeScript 知道 response 不会是 undefined
        const safeResponse = response as DeviceListResponse;

        // 根据当前标签页更新结果
        setQueryState(prev => {
          const newState = { ...prev };
          switch (currentMode) {
            case 'simple':
              newState.simpleParams = {
                ...prev.simpleParams,
                results: {
                  devices: safeResponse.list || [],
                  pagination: {
                    ...prev.simpleParams.results.pagination,
                    total: safeResponse.total || 0,
                  },
                  loading: false,
                  lastUpdated: new Date(),
                }
              };
              break;
            case 'advanced':
              newState.advancedParams = {
                ...prev.advancedParams,
                results: {
                  devices: safeResponse.list || [],
                  pagination: {
                    ...prev.advancedParams.results.pagination,
                    total: safeResponse.total || 0,
                  },
                  loading: false,
                  lastUpdated: new Date(),
                }
              };
              break;
            case 'template':
              newState.templateParams = {
                ...prev.templateParams,
                results: {
                  devices: safeResponse.list || [],
                  pagination: {
                    ...prev.templateParams.results.pagination,
                    total: safeResponse.total || 0,
                  },
                  loading: false,
                  lastUpdated: new Date(),
                }
              };
              break;
          }
          return newState;
        });

        message.success('查询成功');
      }
    } catch (error) {
      console.error('查询失败:', error);
      message.error('查询失败');

      // 根据当前标签页重置加载状态
      setQueryState(prev => {
        const newState = { ...prev };
        switch (currentMode) {
          case 'simple':
            newState.simpleParams = {
              ...prev.simpleParams,
              results: {
                ...prev.simpleParams.results,
                loading: false,
              }
            };
            break;
          case 'advanced':
            newState.advancedParams = {
              ...prev.advancedParams,
              results: {
                ...prev.advancedParams.results,
                loading: false,
              }
            };
            break;
          case 'template':
            newState.templateParams = {
              ...prev.templateParams,
              results: {
                ...prev.templateParams.results,
                loading: false,
              }
            };
            break;
        }
        return newState;
      });
    } finally {
      isExecutingQuery.current = false;
    }
  };

  // 处理刷新按钮点击
  const handleRefresh = () => {
    executeQuery();
  };

  // 当前活动标签页的结果
  const getCurrentResults = () => {
    switch (queryState.mode) {
      case 'simple':
        return queryState.simpleParams.results;
      case 'advanced':
        return queryState.advancedParams.results;
      case 'template':
        return queryState.templateParams.results;
      default:
        return queryState.simpleParams.results;
    }
  };

  // 处理Tab切换
  const handleTabChange = (activeKey: string) => {
    // 更新模式，但保持每个标签页的结果不变
    setQueryState(prev => ({
      ...prev,
      mode: activeKey as 'simple' | 'advanced' | 'template',
    }));
  };

  // 切换到模板标签页
  const switchToTemplateTab = () => {
    setQueryState(prev => ({
      ...prev,
      mode: 'template',
    }));
  };

  // 处理简单查询
  const handleSimpleQuery = () => {
    setQueryState(prev => ({
      ...prev,
      simpleParams: {
        ...prev.simpleParams,
        results: {
          ...prev.simpleParams.results,
          pagination: {
            ...prev.simpleParams.results.pagination,
            current: 1, // 重置到第一页
          },
        }
      }
    }));

    executeQuery();
  };

  // 处理高级查询
  const handleAdvancedQuery = () => {
    // 打印查询条件，用于调试
    console.log('Advanced query groups:', advancedParams.groups);

    setQueryState(prev => ({
      ...prev,
      advancedParams: {
        ...prev.advancedParams,
        results: {
          ...prev.advancedParams.results,
          pagination: {
            ...prev.advancedParams.results.pagination,
            current: 1, // 重置到第一页
          },
        }
      }
    }));

    executeQuery();
  };

  // 更新高级查询条件
  const handleAdvancedQueryChange = (groups: FilterGroup[]) => {
    setQueryState(prev => ({
      ...prev,
      advancedParams: {
        ...prev.advancedParams,
        groups,
      }
    }));
  };

  // 处理模板查询
  const handleTemplateQuery = async (templateId: number, templateName: string) => {
    console.log('handleTemplateQuery called with templateId:', templateId, 'templateName:', templateName);

    // 先更新状态
    setQueryState(prev => ({
      ...prev,
      templateParams: {
        ...prev.templateParams,
        templateId,
        templateName,
        results: {
          ...prev.templateParams.results,
          loading: true, // 设置加载状态
          pagination: {
            ...prev.templateParams.results.pagination,
            current: 1, // 重置到第一页
          },
        }
      }
    }));

    try {
      // 直接获取模板并执行查询，而不是依赖状态更新
      console.log('Fetching template:', templateId);
      const template = await getQueryTemplate(templateId);
      console.log('Template fetched:', template);

      if (template && template.groups) {
        // 确保每个筛选块的 key 和 field 字段同步
        const processedGroups = template.groups.map(group => ({
          ...group,
          blocks: group.blocks.map(block => {
            // 创建一个新的块对象，避免修改原始对象
            const processedBlock = { ...block };

            // 确保 key 和 field 字段同步
            if (processedBlock.field && !processedBlock.key) {
              processedBlock.key = processedBlock.field;
            } else if (processedBlock.key && !processedBlock.field) {
              processedBlock.field = processedBlock.key;
            }

            return processedBlock;
          })
        }));

        const currentPagination = queryState.templateParams.results.pagination;

        console.log('Executing query with processed template groups:', processedGroups);
        const response = await queryDevices({
          groups: processedGroups,
          page: 1, // 始终从第一页开始
          size: currentPagination.pageSize,
        });
        console.log('Query response:', response);

        if (response) {
          // 更新查询结果
          setQueryState(prev => ({
            ...prev,
            templateParams: {
              ...prev.templateParams,
              results: {
                devices: response.list || [],
                pagination: {
                  ...prev.templateParams.results.pagination,
                  current: 1,
                  total: response.total || 0,
                },
                loading: false,
                lastUpdated: new Date(),
              }
            }
          }));

          message.success('查询成功');
        }
      } else {
        message.warning('模板数据不完整');
        // 重置加载状态
        setQueryState(prev => ({
          ...prev,
          templateParams: {
            ...prev.templateParams,
            results: {
              ...prev.templateParams.results,
              loading: false,
            }
          }
        }));
      }
    } catch (error) {
      console.error('查询失败:', error);
      message.error('查询失败');

      // 重置加载状态
      setQueryState(prev => ({
        ...prev,
        templateParams: {
          ...prev.templateParams,
          results: {
            ...prev.templateParams.results,
            loading: false,
          }
        }
      }));
    }
  };

  // 删除模板
  const handleDeleteTemplate = async (templateId: number, templateName: string) => {
    Modal.confirm({
      title: '删除查询模板',
      content: `确定要删除模板「${templateName}」吗？此操作不可恢复。`,
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        try {
          await deleteQueryTemplate(templateId);
          message.success(`模板「${templateName}」已删除`);

          // 刷新模板列表
          await loadTemplates();

          // 如果当前正在使用该模板进行查询，则重置查询状态
          if (queryState.mode === 'template' && queryState.templateParams.templateId === templateId) {
            setQueryState(prev => ({
              ...prev,
              templateParams: {
                ...prev.templateParams,
                templateId: null,
                templateName: '',
              },
            }));
          }
        } catch (error) {
          console.error('删除模板失败:', error);
          message.error('删除模板失败');
        }
      },
    });
  };

  // 加载模板到高级查询
  const handleEditTemplate = async (templateId: number) => {
    try {
      const template = await getQueryTemplate(templateId);
      if (template && template.groups) {
        // 处理模板数据，确保每个筛选块都有key字段
        const processedGroups = template.groups.map(group => ({
          ...group,
          blocks: group.blocks.map(block => {
            // 如果有field但没有key，将field的值赋给key
            if (block.field && !block.key) {
              return { ...block, key: block.field };
            }
            // 如果有key但没有field，将key的值赋给field
            if (block.key && !block.field) {
              return { ...block, field: block.key };
            }
            return block;
          })
        }));

        // 切换到高级查询模式
        setQueryState(prev => ({
          ...prev,
          mode: 'advanced',
          advancedParams: {
            ...prev.advancedParams,
            groups: processedGroups,
            sourceTemplateId: templateId,  // 添加模板来源ID
            sourceTemplateName: template.name  // 添加模板来源名称
          }
        }));

        message.success(`已加载模板「${template.name}」到高级查询，可以进行编辑`);
      } else {
        message.warning('模板数据不完整');
      }
    } catch (error) {
      console.error('加载模板失败:', error);
      message.error('加载模板失败');
    }
  };

  // 处理分页变化
  const handlePaginationChange = async (page: number, pageSize?: number) => {
    // 根据当前标签页更新分页信息
    switch (queryState.mode) {
      case 'simple':
        setQueryState(prev => ({
          ...prev,
          simpleParams: {
            ...prev.simpleParams,
            results: {
              ...prev.simpleParams.results,
              pagination: {
                ...prev.simpleParams.results.pagination,
                current: page,
                pageSize: pageSize || prev.simpleParams.results.pagination.pageSize,
              },
            }
          }
        }));
        break;
      case 'advanced':
        setQueryState(prev => ({
          ...prev,
          advancedParams: {
            ...prev.advancedParams,
            results: {
              ...prev.advancedParams.results,
              pagination: {
                ...prev.advancedParams.results.pagination,
                current: page,
                pageSize: pageSize || prev.advancedParams.results.pagination.pageSize,
              },
            }
          }
        }));
        break;
      case 'template':
        setQueryState(prev => ({
          ...prev,
          templateParams: {
            ...prev.templateParams,
            results: {
              ...prev.templateParams.results,
              pagination: {
                ...prev.templateParams.results.pagination,
                current: page,
                pageSize: pageSize || prev.templateParams.results.pagination.pageSize,
              },
            }
          }
        }));

        // 对于模板查询，直接执行查询而不是调用executeQuery
        if (queryState.templateParams.templateId) {
          try {
            const template = await getQueryTemplate(queryState.templateParams.templateId);
            if (template && template.groups) {
              const response = await queryDevices({
                groups: template.groups,
                page: page,
                size: pageSize || queryState.templateParams.results.pagination.pageSize,
              });

              if (response) {
                setQueryState(prev => ({
                  ...prev,
                  templateParams: {
                    ...prev.templateParams,
                    results: {
                      devices: response.list || [],
                      pagination: {
                        ...prev.templateParams.results.pagination,
                        current: page,
                        total: response.total || 0,
                      },
                      loading: false,
                      lastUpdated: new Date(),
                    }
                  }
                }));
              }
            }
          } catch (error) {
            console.error('分页查询失败:', error);
            message.error('分页查询失败');
          }
          return; // 不执行下面的executeQuery()
        }
        break;
    }

    // 对于简单查询和高级查询，使用executeQuery
    if (queryState.mode !== 'template') {
      executeQuery();
    }
  };

  // 同步设备信息（前端实现，不需要后端）
  const handleSync = () => {
    // 显示加载中消息
    const hide = message.loading('正在同步设备信息...', 0);

    // 模拟同步过程
    setTimeout(() => {
      hide();
      message.success('设备信息同步成功');
      // 刷新当前设备列表
      executeQuery();
    }, 2000); // 模拟2秒的同步时间
  };

  // 获取机器用途选项，返回 Promise 以便在 handleRoleEdit 中使用 then 方法
  const fetchGroupOptions = async (): Promise<void> => {
    // 移除条件判断，每次打开对话框都重新获取选项
    try {
      setLoadingGroupOptions(true);
      const values = await getDeviceFieldValues('group', 10000);
      console.log('Fetched group options:', values); // 调试日志

      // 处理选项，如果包含分号，只取分号前的部分
      const processedValues = values.map(value => {
        if (value && value.includes(';')) {
          return value.split(';')[0];
        }
        return value;
      });

      // 去除重复项和空值
      const uniqueValues = Array.from(new Set(processedValues.filter(v => v && v.trim() !== '')));
      console.log('Processed group options:', uniqueValues); // 调试日志
      setGroupOptions(uniqueValues);
    } catch (error) {
      console.error('获取机器用途选项失败:', error);
      message.error('获取机器用途选项失败');
    } finally {
      setLoadingGroupOptions(false);
    }
  };

  // 处理用途编辑
  const handleRoleEdit = (device: Device) => {
    console.log('Editing device:', device); // 调试日志
    setEditingDevice(device);

    // 先获取机器用途选项，确保选项列表已加载
    fetchGroupOptions().then(() => {
      // 如果用途包含分号，只取分号前的部分（原始的用途值）
      let originalGroup = device.group;
      if (originalGroup && originalGroup.includes(';')) {
        originalGroup = originalGroup.split(';')[0];
      }

      console.log('Setting form value:', originalGroup); // 调试日志

      // 在 tags 模式下，需要设置数组值，确保当前用途被选中
      // 使用 setTimeout 确保在对话框渲染后设置表单值
      setTimeout(() => {
        roleForm.setFieldsValue({ group: originalGroup ? [originalGroup] : [] });
      }, 100);

      setRoleEditVisible(true);
    });
  };

  // 保存用途编辑
  const handleRoleSave = async () => {
    try {
      const values = await roleForm.validateFields();
      if (!editingDevice) return;

      // 处理 tags 模式下的值，只取第一个值
      let groupValue = values.group;
      if (Array.isArray(groupValue)) {
        if (groupValue.length > 0) {
          groupValue = groupValue[0];

          // 如果用户输入的值包含分号，则去除分号及其后面的内容
          if (groupValue.includes(';')) {
            groupValue = groupValue.split(';')[0];
          }
        } else {
          // 空数组表示空值
          groupValue = '';
        }
      }
      // 允许 groupValue 为空

      await updateDeviceGroup(editingDevice.id, groupValue);
      message.success('用途更新成功');

      // 重新从后端获取数据，确保显示最新的数据
      console.log('重新从后端获取数据...');
      executeQuery();

      // 如果是新的用途值，添加到选项中
      if (groupValue && !groupOptions.includes(groupValue)) {
        setGroupOptions(prev => [...prev, groupValue]);
      }

      setRoleEditVisible(false);
    } catch (error) {
      console.error('用途更新失败:', error);
      message.error('用途更新失败');
    }
  };

  // 表格列定义
  const columns: ColumnsType<Device> = [
    {
      title: '设备编码',
      dataIndex: 'ciCode',
      key: 'ciCode',
      render: (text: string, record: Device) => (
        <Button
          type="link"
          onClick={() => navigate(`/device/${record.id}/detail`)}
          style={{ padding: 0 }}
        >
          {text}
        </Button>
      ),
    },
    {
      title: 'IP地址',
      dataIndex: 'ip',
      key: 'ip',
    },
    {
      title: '机器用途',
      dataIndex: 'group',
      key: 'group',
      render: (text: string, record: Device) => {
        // 从第6个字符开始渐变，最多显示10个字符
        const gradientThreshold = 6; // 从第6个字符开始渐变

        // 组合机器用途和应用名称
        let displayText = text || '';
        if (!displayText && record.appName) {
          // 如果机器用途为空但有应用名称，直接显示应用名称
          displayText = record.appName;
        } else if (displayText && record.appName) {
          // 如果机器用途和应用名称都有，用分号连接
          displayText = `${displayText};${record.appName}`;
        }

        // 如果文本长度超过渐变阈值，则显示渐变效果
        const hasMore = displayText && displayText.length > gradientThreshold;

        // 根据记录的特性确定样式类
        let styleClass = 'group-text';
        if (hasMore) styleClass += ' has-more';
        if (record.isSpecial) styleClass += ' special-bg';
        else if (record.appName && record.appName.trim() !== '') styleClass += ' special-bg';
        else if (record.cluster && record.cluster.trim() !== '') styleClass += ' cluster-bg';

        return (
          <Space>
            <span className={styleClass}>
              {displayText}
            </span>
            <Tooltip title="编辑用途">
              <Button
                type="link"
                icon={<EditOutlined />}
                size="small"
                onClick={(e) => {
                  e.stopPropagation();
                  handleRoleEdit(record);
                }}
              />
            </Tooltip>
          </Space>
        );
      },
    },
    {
      title: '所属集群',
      dataIndex: 'cluster',
      key: 'cluster',
    },
    {
      title: '集群角色',
      dataIndex: 'role',
      key: 'role',
    },
    {
      title: 'CPU架构',
      dataIndex: 'archType',
      key: 'archType',
    },
    {
      title: 'IDC',
      dataIndex: 'idc',
      key: 'idc',
    },
    {
      title: 'ROOM',
      dataIndex: 'room',
      key: 'room',
    },
    {
      title: '网络区域',
      dataIndex: 'netZone',
      key: 'netZone',
    },
    {
      title: 'APPID',
      dataIndex: 'appId',
      key: 'appId',
    },
    {
      title: '是否国产化',
      dataIndex: 'isLocalization',
      key: 'isLocalization',
      render: (value: boolean) => (
        <Tag color={value ? 'green' : 'default'}>
          {value ? '是' : '否'}
        </Tag>
      ),
    },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: Device) => (
        <Button
          type="link"
          onClick={() => navigate(`/device/${record.id}/detail`)}
          style={{ padding: 0 }}
        >
          详情
        </Button>
      ),
    },
  ];

  const {
    mode,
    simpleParams,
    advancedParams,
    templateParams,
  } = queryState;

  // 获取当前标签页的结果
  const results = getCurrentResults();
  const { devices, pagination, loading, lastUpdated } = results;

  // 渲染模板列表
  const renderTemplateList = () => {
    // 搜索框
    const renderSearchHeader = () => (
      <div style={{ display: 'flex', justifyContent: 'flex-end', alignItems: 'center', marginBottom: 16 }}>
        <Input.Search
          placeholder="搜索模板名称或描述"
          allowClear
          style={{ width: 300 }}
          value={templateSearchKeyword}
          onChange={(e) => setTemplateSearchKeyword(e.target.value)}
          onSearch={handleTemplateSearch}
        />
      </div>
    );

    if (templates.length === 0 && templatePagination.total === 0) {
      return (
        <>
          {renderSearchHeader()}
          <div className="empty-template-container">
            <div className="empty-template-icon">
              <FileSearchOutlined />
            </div>
            <div className="empty-template-title">暂无查询模板</div>
            <div className="empty-template-description">
              您可以在高级查询中创建查询条件，然后将其保存为模板以便于快速复用。
              模板可以帮助您更高效地管理常用查询，提高工作效率。
            </div>
            <div className="empty-template-actions">
              <Button
                type="primary"
                size="large"
                icon={<PlusOutlined />}
                onClick={() => {
                  // 切换到高级查询标签页
                  setQueryState(prev => ({
                    ...prev,
                    mode: 'advanced',
                  }));
                }}
              >
                创建新查询
              </Button>
            </div>
          </div>
        </>
      );
    }

    // 直接使用模板列表和分页信息
    const { current, pageSize, total } = templatePagination;

    return (
      <>
        {renderSearchHeader()}

        {templates.length === 0 && templateSearchKeyword ? (
          <div className="empty-template-container" style={{ minHeight: '200px' }}>
            <div className="empty-template-icon">
              <FileSearchOutlined />
            </div>
            <div className="empty-template-title">没有找到匹配的模板</div>
            <div className="empty-template-description">
              请尝试使用其他关键词进行搜索，或清除搜索条件查看所有模板。
              <div style={{ marginTop: '12px', textAlign: 'center' }}>
                <Button
                  type="primary"
                  onClick={() => handleTemplateSearch('')}
                  icon={<ReloadOutlined />}
                >
                  显示所有模板
                </Button>
              </div>
            </div>
          </div>
        ) : (
          <>
            <div className="template-list">
              {templates.map(template => {
                // 生成条件组的缩略信息
                const querySummary = generateQuerySummary(template.groups || [], 200);

                return (
                  <Card
                    key={template.id}
                    title={<span style={{ fontWeight: 600, fontSize: '16px', color: '#1890ff' }}>{template.name}</span>}
                    className="template-card"
                    extra={
                      <Space>
                        <Tooltip title="编辑查询参数">
                          <Button
                            type="default"
                            size="small"
                            icon={<EditOutlined />}
                            onClick={() => handleEditTemplate(template.id)}
                          />
                        </Tooltip>
                        <Tooltip title="执行查询">
                          <Button
                            type="primary"
                            size="small"
                            icon={<PlayCircleOutlined />}
                            onClick={() => handleTemplateQuery(template.id, template.name)}
                          >
                            执行
                          </Button>
                        </Tooltip>
                        <Tooltip title="删除模板">
                          <Button
                            type="default"
                            danger
                            size="small"
                            icon={<DeleteOutlined />}
                            onClick={(e) => {
                              e.stopPropagation();
                              handleDeleteTemplate(template.id, template.name);
                            }}
                          />
                        </Tooltip>
                      </Space>
                    }
                  >
                    {template.description && (
                      <Paragraph style={{
                        marginBottom: 12,
                        padding: '8px 10px',
                        background: 'rgba(0, 0, 0, 0.02)',
                        borderRadius: '6px',
                        border: '1px dashed rgba(0, 0, 0, 0.09)'
                      }}>
                        <strong style={{ color: '#555' }}>描述：</strong>{template.description}
                      </Paragraph>
                    )}

                    <Paragraph style={{
                      marginBottom: 12,
                      display: 'inline-block',
                      padding: '4px 10px',
                      background: 'rgba(82, 196, 26, 0.1)',
                      borderRadius: '12px',
                      color: '#52c41a',
                      fontWeight: 500
                    }}>
                      <strong>条件组：</strong>{template.groups?.length || 0}个
                    </Paragraph>

                    <Paragraph
                      ellipsis={{ rows: 2, expandable: true, symbol: '展开' }}
                      style={{
                        background: 'rgba(24, 144, 255, 0.05)',
                        padding: '12px',
                        borderRadius: '8px',
                        marginBottom: 0,
                        border: '1px solid rgba(24, 144, 255, 0.1)',
                        boxShadow: '0 1px 2px rgba(0, 0, 0, 0.03)'
                      }}
                    >
                      <strong style={{ color: '#1890ff' }}>查询条件：</strong>{querySummary}
                    </Paragraph>
                  </Card>
                );
              })}
            </div>

            {/* 分页器 */}
            <div className="template-pagination">
              <Pagination
                current={current}
                pageSize={pageSize}
                total={total}
                onChange={handleTemplatePageChange}
                showSizeChanger
                pageSizeOptions={['4', '8', '12', '16']}
                showTotal={(total) => `共 ${total} 个模板`}
              />
            </div>
          </>
        )}
      </>
    );
  };

  return (
    <div className="device-center-container">
      <Card
        title={
          <div style={{ display: 'flex', alignItems: 'center' }}>
            <DatabaseOutlined style={{ marginRight: '12px', color: '#1677ff', fontSize: '18px' }} />
            <span>设备中心</span>
          </div>
        }
        extra={
          <div></div>
        }
        className="device-center-card"
      >
        {initialParams.templateId && !initialLoadComplete ? (
          // 如果是从模板预览跳转过来且初始加载未完成，显示加载状态
          <div style={{ padding: '40px 0', textAlign: 'center' }}>
            <Spin size="large" tip="正在加载模板数据..." />
          </div>
        ) : (
          <Tabs
            activeKey={mode}
            onChange={handleTabChange}
            animated={{ inkBar: true, tabPane: false }} // 只启用墨条动画，禁用内容切换动画
            destroyInactiveTabPane={false} // 不销毁不活动的标签页，避免重新渲染
          >
            <TabPane tab="基本查询" key="simple" forceRender>
              <SimpleQueryPanel
                keyword={simpleParams.keyword}
                onKeywordChange={(keyword) =>
                  setQueryState(prev => ({
                    ...prev,
                    simpleParams: {
                      ...prev.simpleParams,
                      keyword
                    }
                  }))
                }
                onSearch={handleSimpleQuery}
                loading={loading}
              />
            </TabPane>
            <TabPane tab="高级查询" key="advanced" forceRender>
              <AdvancedQueryPanel
                filterGroups={advancedParams.groups}
                onFilterGroupsChange={handleAdvancedQueryChange}
                onQuery={handleAdvancedQuery}
                loading={loading}
                sourceTemplateId={advancedParams.sourceTemplateId}
                sourceTemplateName={advancedParams.sourceTemplateName}
                onTemplateSaved={loadTemplates} // 保存模板后刷新模板列表
                onSwitchToTemplateTab={switchToTemplateTab} // 切换到模板标签页
              />
            </TabPane>
            <TabPane tab="模板" key="template" forceRender>
              {renderTemplateList()}
            </TabPane>
          </Tabs>
        )}

        {/* 查询摘要 */}
        <QuerySummary
          mode={mode}
          simpleKeyword={simpleParams.keyword}
          advancedGroups={advancedParams.groups}
          templateName={templateParams.templateName}
          resultCount={pagination.total}
          lastUpdated={lastUpdated}
        />

        {/* 只有当有查询结果时才显示设备列表和操作按钮 */}
        {devices.length > 0 && (
          <>
            {/* 设备列表标题和操作按钮 */}
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
              <div>
                <Typography.Title level={5} style={{ margin: 0 }}>设备列表</Typography.Title>
              </div>
              <Space>
                <Tooltip title="刷新数据">
                  <Button
                    icon={<ReloadOutlined />}
                    onClick={handleRefresh}
                    loading={loading}
                  >
                    刷新
                  </Button>
                </Tooltip>
                <Tooltip title="同步设备信息">
                  <Button
                    icon={<SyncOutlined />}
                    onClick={handleSync}
                  >
                    同步
                  </Button>
                </Tooltip>
              </Space>
            </div>

            {/* 设备列表 */}
            <Table
              columns={columns}
              dataSource={devices}
              rowKey="id"
              pagination={{
                current: pagination.current,
                pageSize: pagination.pageSize,
                total: pagination.total,
                onChange: handlePaginationChange,
                showSizeChanger: true,
                showQuickJumper: true,
              }}
              loading={loading}
              onRow={(record) => {
                // 根据条件决定背景色
                let bgColor = '';
                if (record.isSpecial) {
                  // 浅黄色背景 - 特殊设备
                  bgColor = '#fffbe6';
                } else if (record.appName && record.appName.trim() !== '') {
                  // 浅黄色背景 - 应用名称不为空
                  bgColor = '#fffbe6';
                } else if (record.cluster && record.cluster.trim() !== '') {
                  // 浅绿色背景 - 集群不为空且非特殊设备
                  bgColor = '#f6ffed';
                }

                return {
                  style: { backgroundColor: bgColor },
                  onMouseEnter: async (e) => {
                    // 检查当前鼠标是否在"详情"按钮上或附近
                    const target = e.target as HTMLElement;
                    const isOnDetailButton = target.tagName === 'BUTTON' ||
                                            target.tagName === 'SPAN' ||
                                            target.closest('button') !== null;

                    // 检查当前URL是否包含详情页路径
                    const isDetailPage = window.location.pathname.includes(`/device/${record.id}/detail`) ||
                                         window.location.pathname.includes('/detail') ||
                                         (window.location.pathname.includes('/device/') && window.location.pathname.endsWith('/detail'));

                    // 如果是特殊设备或有应用名称，且不在详情页且不在"详情"按钮上，显示提示框
                    if ((record.isSpecial || (record.appName && record.appName.trim() !== '')) && !isDetailPage && !isOnDetailButton) {
                      try {
                        // 创建提示框
                        const tooltip = document.createElement('div');
                        tooltip.className = 'device-feature-tooltip';

                        // 初始内容 - 美化样式
                        tooltip.innerHTML = `
                          <div style="font-weight: 600; font-size: 14px; color: #1677ff; margin-bottom: 8px; border-bottom: 1px solid rgba(0,0,0,0.06); padding-bottom: 6px;">
                            <span style="margin-right: 5px;">&#x1F4CB;</span>设备特性
                          </div>
                          <div style="color: #666; font-size: 13px;">
                            <div style="display: flex; align-items: center;">
                              <span style="margin-right: 5px;">&#x231B;</span>正在加载特性详情...
                            </div>
                          </div>
                        `;

                        // 计算位置 - 跟随鼠标
                        tooltip.style.position = 'fixed';
                        tooltip.style.left = `${e.clientX + 15}px`; // 鼠标右侧偏移15px
                        tooltip.style.top = `${e.clientY - 20}px`;  // 鼠标上方偏移20px

                        // 美化样式 - 半透明卡片
                        tooltip.style.backgroundColor = 'rgba(255, 255, 255, 0.9)';
                        tooltip.style.backdropFilter = 'blur(5px)'; // 模糊效果
                        tooltip.style.border = '1px solid rgba(217, 217, 217, 0.6)';
                        tooltip.style.borderRadius = '8px';
                        tooltip.style.padding = '12px';
                        tooltip.style.boxShadow = '0 4px 12px rgba(0, 0, 0, 0.08)';
                        tooltip.style.zIndex = '1000';
                        tooltip.style.maxWidth = '350px';
                        tooltip.style.transition = 'all 0.2s ease-in-out';

                        document.body.appendChild(tooltip);

                        // 存储提示框引用，便于移除
                        e.currentTarget.tooltip = tooltip;

                        // 创建一个引用副本，避免使用 e.currentTarget
                        const tooltipElement = tooltip;
                        const rowElement = e.currentTarget;

                        // 添加鼠标移动事件，使提示框跟随鼠标
                        const handleMouseMove = (moveEvent: MouseEvent) => {
                          // 检查当前URL是否包含详情页路径
                          const isDetailPage = window.location.pathname.includes(`/device/${record.id}/detail`) ||
                                               window.location.pathname.includes('/detail') ||
                                               (window.location.pathname.includes('/device/') && window.location.pathname.endsWith('/detail'));

                          // 如果在详情页面上，隐藏提示框并返回
                          if (isDetailPage && tooltipElement && document.body.contains(tooltipElement)) {
                            tooltipElement.style.display = 'none';
                            return;
                          }

                          // 检查当前鼠标是否在"详情"按钮上或附近
                          const target = moveEvent.target as HTMLElement;
                          const isOnDetailButton = target.tagName === 'BUTTON' ||
                                                  target.tagName === 'SPAN' ||
                                                  target.closest('button') !== null;

                          // 如果鼠标在"详情"按钮上，隐藏提示框
                          if (isOnDetailButton && tooltipElement && document.body.contains(tooltipElement)) {
                            tooltipElement.style.display = 'none';
                            return;
                          } else if (tooltipElement && !isDetailPage) {
                            tooltipElement.style.display = 'block';
                          }

                          // 检查元素和提示框是否仍然存在
                          if (rowElement && rowElement.tooltip && tooltipElement && document.body.contains(tooltipElement)) {
                            // 更新提示框位置
                            tooltipElement.style.left = `${moveEvent.clientX + 15}px`;
                            tooltipElement.style.top = `${moveEvent.clientY - 20}px`;

                            // 检查是否超出右边界
                            const tooltipRect = tooltipElement.getBoundingClientRect();
                            const windowWidth = window.innerWidth;
                            if (tooltipRect.right > windowWidth) {
                              // 如果超出右边界，则将提示框放在鼠标左侧
                              tooltipElement.style.left = `${moveEvent.clientX - tooltipRect.width - 15}px`;
                            }
                          } else {
                            // 如果元素或提示框不存在，移除事件监听器
                            document.removeEventListener('mousemove', handleMouseMove);
                          }
                        };

                        // 将鼠标移动事件处理函数存储在元素上，便于移除
                        e.currentTarget.handleMouseMove = handleMouseMove;
                        document.addEventListener('mousemove', handleMouseMove);

                        // 构建特性详情数组
                        const featureDetails: string[] = [];

                        // 添加机器用途
                        if (record.group && record.group.trim() !== '') {
                          featureDetails.push(`机器用途: ${record.group}`);
                        }

                        // 添加应用名称
                        if (record.appName && record.appName.trim() !== '') {
                          featureDetails.push(`应用名称: ${record.appName}`);
                        }

                        // 如果有标签特性或污点特性，获取详情
                        if (record.featureCount && record.featureCount > (record.group ? 1 : 0)) {
                          // 获取设备特性详情
                          const details = await getDeviceFeatureDetails(record.ciCode);

                          // 添加标签详情
                          if (details.labels && details.labels.length > 0) {
                            featureDetails.push(`存在标签:`);
                            details.labels.forEach(label => {
                              featureDetails.push(`  ${label.key}=${label.value}`);
                            });
                          }

                          // 添加污点详情
                          if (details.taints && details.taints.length > 0) {
                            featureDetails.push(`存在污点:`);
                            details.taints.forEach(taint => {
                              featureDetails.push(`  ${taint.key}=${taint.value}:${taint.effect}`);
                            });
                          }
                        }

                        // 更新提示框内容 - 美化样式
                        let tooltipContent = `
                          <div style="font-weight: 600; font-size: 14px; color: #1677ff; margin-bottom: 8px; border-bottom: 1px solid rgba(0,0,0,0.06); padding-bottom: 6px;">
                            <span style="margin-right: 5px;">&#x1F4CB;</span>设备特性
                          </div>
                          <div style="color: #666; font-size: 13px;">
                        `;

                        // 添加机器用途信息
                        const groupInfo = featureDetails.find(detail => detail.startsWith('机器用途:'));
                        if (groupInfo) {
                          const groupValue = groupInfo.split(':')[1].trim();
                          tooltipContent += `
                            <div style="margin-bottom: 12px;">
                              <div style="display: flex; align-items: center;">
                                <span style="margin-right: 5px; color: #1677ff;">&#x1F4BB;</span>
                                <span style="font-weight: 500;">机器用途:</span>
                              </div>
                              <div>
                                <span style="background-color: rgba(22, 119, 255, 0.1); padding: 2px 8px; border-radius: 4px; color: #1677ff; display: inline-block; text-align: left;">${groupValue.trim()}</span>
                              </div>
                            </div>
                          `;
                        }

                        // 添加应用名称信息
                        const appNameInfo = featureDetails.find(detail => detail.startsWith('应用名称:'));
                        if (appNameInfo) {
                          const appNameValue = appNameInfo.split(':')[1].trim();
                          tooltipContent += `
                            <div style="margin-bottom: 12px;">
                              <div style="display: flex; align-items: center;">
                                <span style="margin-right: 5px; color: #722ed1;">&#x1F4F1;</span>
                                <span style="font-weight: 500;">应用名称:</span>
                              </div>
                              <div>
                                <span style="background-color: rgba(114, 46, 209, 0.1); padding: 2px 8px; border-radius: 4px; color: #722ed1; display: inline-block; text-align: left;">${appNameValue.trim()}</span>
                              </div>
                            </div>
                          `;
                        }

                        // 添加标签信息
                        const labelIndex = featureDetails.findIndex(detail => detail === '存在标签:');
                        if (labelIndex !== -1) {
                          tooltipContent += `
                            <div style="margin-bottom: 8px;">
                              <div style="display: flex; align-items: center;">
                                <span style="margin-right: 5px; color: #52c41a;">&#x1F3F7;</span>
                                <span style="font-weight: 500;">标签信息:</span>
                              </div>
                            </div>
                            <div style="margin-bottom: 12px;">
                          `;

                          // 收集标签详情
                          let i = labelIndex + 1;
                          while (i < featureDetails.length && featureDetails[i].startsWith('  ')) {
                            const labelParts = featureDetails[i].trim().split('=');
                            if (labelParts.length === 2) {
                              tooltipContent += `
                                <div style="margin-bottom: 8px;">
                                  <div style="color: #666; font-weight: 500;">${labelParts[0]}:</div>
                                  <div>
                                    <span style="background-color: rgba(82, 196, 26, 0.1); padding: 2px 8px; border-radius: 4px; color: #52c41a; display: inline-block; text-align: left;">${labelParts[1].trim()}</span>
                                  </div>
                                </div>
                              `;
                            }
                            i++;
                          }

                          tooltipContent += `</div>`;
                        }

                        // 添加污点信息
                        const taintIndex = featureDetails.findIndex(detail => detail === '存在污点:');
                        if (taintIndex !== -1) {
                          tooltipContent += `
                            <div style="margin-bottom: 8px;">
                              <div style="display: flex; align-items: center;">
                                <span style="margin-right: 5px; color: #f5222d;">&#x26A0;</span>
                                <span style="font-weight: 500;">污点信息:</span>
                              </div>
                            </div>
                            <div style="margin-bottom: 12px;">
                          `;

                          // 收集污点详情
                          let i = taintIndex + 1;
                          while (i < featureDetails.length && featureDetails[i].startsWith('  ')) {
                            const taintInfo = featureDetails[i].trim();
                            const taintParts = taintInfo.split('=');
                            if (taintParts.length === 2) {
                              const keyPart = taintParts[0];
                              const valueParts = taintParts[1].split(':');
                              if (valueParts.length === 2) {
                                const valuePart = valueParts[0];
                                const effectPart = valueParts[1];

                                // 根据 effect 类型选择颜色
                                let effectColor = '#f5222d'; // 默认红色
                                if (effectPart === 'PreferNoSchedule') {
                                  effectColor = '#faad14'; // 黄色
                                } else if (effectPart === 'NoExecute') {
                                  effectColor = '#f5222d'; // 红色
                                }

                                tooltipContent += `
                                  <div style="margin-bottom: 12px;">
                                    <div style="color: #666; font-weight: 500;">${keyPart}:</div>
                                    <div>
                                      <div>
                                        <span style="background-color: rgba(245, 34, 45, 0.1); padding: 2px 8px; border-radius: 4px; color: #f5222d; display: inline-block; text-align: left;">${valuePart.trim()}</span>
                                      </div>
                                      <div>
                                        <span style="background-color: rgba(0, 0, 0, 0.04); padding: 2px 8px; border-radius: 4px; color: ${effectColor}; font-size: 12px; display: inline-block; text-align: left;">${effectPart.trim()}</span>
                                      </div>
                                    </div>
                                  </div>
                                `;
                              }
                            }
                            i++;
                          }

                          tooltipContent += `</div>`;
                        }

                        tooltipContent += `</div>`;
                        tooltip.innerHTML = tooltipContent;
                      } catch (error) {
                        console.error('获取设备特性详情失败:', error);
                      }
                    }
                  },
                  onMouseLeave: (e) => {
                    try {
                      // 安全地移除提示框
                      const tooltip = e.currentTarget?.tooltip;
                      if (tooltip && document.body.contains(tooltip)) {
                        document.body.removeChild(tooltip);
                      }

                      // 清除引用
                      if (e.currentTarget) {
                        e.currentTarget.tooltip = null;
                      }

                      // 安全地移除鼠标移动事件监听器
                      const handleMouseMove = e.currentTarget?.handleMouseMove;
                      if (handleMouseMove) {
                        document.removeEventListener('mousemove', handleMouseMove);

                        // 清除引用
                        if (e.currentTarget) {
                          e.currentTarget.handleMouseMove = null;
                        }
                      }
                    } catch (error) {
                      console.error('清除提示框资源失败:', error);
                    }
                  }
                };
              }}
            />
          </>
        )}
      </Card>

      {/* 用途编辑对话框 */}
      <Modal
        title="编辑机器用途"
        open={roleEditVisible}
        onOk={handleRoleSave}
        onCancel={() => {
          setRoleEditVisible(false);
          roleForm.resetFields(); // 重置表单
        }}
        afterClose={() => {
          roleForm.resetFields(); // 关闭后重置表单
        }}
        destroyOnClose
        className="group-edit-modal"
        centered
      >
        <Form form={roleForm} layout="vertical" preserve={false}>
          <Form.Item
            name="group"
            label="机器用途"
            rules={[]}
            help="可以设置为空值"
          >
            <Select<string[], { value: string; children: React.ReactNode }>
              showSearch
              allowClear
              loading={loadingGroupOptions}
              placeholder="请选择或输入机器用途，清空表示无用途"
              style={{ width: '100%' }}
              notFoundContent={loadingGroupOptions ? <Spin size="small" /> : null}
              mode="tags"
              maxTagCount={1}
              tokenSeparators={[',']}
              optionFilterProp="children"
              filterOption={(input, option) =>
                (option?.children as unknown as string)?.toLowerCase().includes(input.toLowerCase())
              }
              getPopupContainer={() => document.body}
              dropdownStyle={{ maxHeight: '300px', overflow: 'auto' }}
              virtual={true}
              listHeight={256}
              placement="bottomLeft"
              popupMatchSelectWidth={false}
              className="group-select"
            >
              {groupOptions.map(group => (
                <Option key={group} value={group}>{group}</Option>
              ))}
            </Select>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default DeviceCenter;
