// Operations management types

export interface OpsJob {
  id: number;
  name: string;
  description: string;
  status: string;
  progress: number;
  start_time: string;
  end_time: string;
  log_content?: string;
  created_at: string;
  updated_at: string;
}

export interface OpsJobListResponse {
  list: OpsJob[];
  page: number;
  size: number;
  total: number;
}

export interface OpsJobQuery {
  page?: number;
  size?: number;
  name?: string;
  status?: string;
}

export interface OpsJobCreateDTO {
  name: string;
  description: string;
}

export interface OpsJobStatusUpdate {
  id: number;
  status: string;
  progress: number;
  message: string;
  log_line?: string;
}
