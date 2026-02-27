import type {
    AIServiceConfig,
    AIServiceType,
    CreateAIConfigRequest,
    TestConnectionRequest,
    UpdateAIConfigRequest
} from '../types/ai'
import request from '../utils/request'

export interface DefaultAIConfig {
  model: string
  provider?: string
  name?: string
}

export const aiAPI = {
  list(serviceType?: AIServiceType) {
    return request.get<AIServiceConfig[]>('/ai-configs', {
      params: { service_type: serviceType }
    })
  },

  /** 获取当前默认配置（与后端生图/生视频使用的逻辑一致） */
  getDefault(serviceType: AIServiceType) {
    return request.get<DefaultAIConfig>('/ai-configs/default', {
      params: { service_type: serviceType }
    })
  },

  create(data: CreateAIConfigRequest) {
    return request.post<AIServiceConfig>('/ai-configs', data)
  },

  get(id: number) {
    return request.get<AIServiceConfig>(`/ai-configs/${id}`)
  },

  update(id: number, data: UpdateAIConfigRequest) {
    return request.put<AIServiceConfig>(`/ai-configs/${id}`, data)
  },

  delete(id: number) {
    return request.delete(`/ai-configs/${id}`)
  },

  testConnection(data: TestConnectionRequest) {
    return request.post('/ai-configs/test', data)
  },

  extractStyle(episodeId: string) {
    return request.post(`/episodes/${episodeId}/style/extract`)
  }
}
