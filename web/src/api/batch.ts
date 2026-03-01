import request from '../utils/request'

export interface BatchImageProgress {
  expected_total: number
  completed: number
  failed: number
  pending: number
  processing: number
}

export interface BatchGenerateFramesResult {
  total: number
  success: number
  failed: number
  failed_storyboard_ids?: number[]
}

export interface BatchGenerateVideosResult {
  total: number
  success: number
  failed: number
  failed_storyboard_ids?: number[]
}

export const batchAPI = {
  /** 一键生图：为当前剧集所有分镜生成首尾帧，返回 task_id，需轮询 /tasks/:id 获取进度与结果 */
  generateFrames(episodeId: string) {
    return request.post<{ task_id: string }>('/batch/generate-frames', { episode_id: episodeId })
  },

  /** 重试失败分镜：为失败分镜重新生成首尾帧，返回 task_id，需轮询 /tasks/:id 获取进度与结果 */
  retryFailedFrames(episodeId: string, failedStoryboardIds: number[]) {
    return request.post<{ task_id: string }>('/batch/retry-failed-frames', {
      episode_id: episodeId,
      failed_storyboard_ids: failedStoryboardIds,
    })
  },

  /** 恢复一键生图任务（继续执行失败的分镜） */
  resumeBatchFrames(data: {
    task_id: string
    failed_storyboard_ids: number[]
  }) {
    return request.post<{ task_id: string }>('/batch/resume-batch-frames', data)
  },

  /** 一键出片：为当前剧集所有分镜用首尾帧生成视频，返回 task_id */
  generateVideos(episodeId: string, model: string) {
    return request.post<{ task_id: string }>('/batch/generate-videos', {
      episode_id: episodeId,
      model,
    })
  },

  /** 剧集首尾帧生图进度（只读），供前端轮询真实进度 */
  getBatchImageProgress(episodeId: string) {
    return request.get<BatchImageProgress>(`/episodes/${episodeId}/batch-image-progress`)
  },

  /** 一键章节视频：生图→出片→合成，返回 task_id */
  generateEpisodeVideo(data: {
    episode_id: string
    drama_id?: string
    model: string
  }) {
    return request.post<{ task_id: string }>('/batch/generate-episode-video', data)
  },

  /** 重试一键章节视频的某个阶段 */
  retryEpisodeVideoPhase(data: {
    task_id: string
    phase: 'frames' | 'videos' | 'merge'
    reset_progress?: boolean  // true: 全部重试（重置进度），false: 继续执行（从失败处继续）
  }) {
    return request.post<{ task_id: string }>('/batch/retry-episode-video-phase', data)
  },

  /** 取消一键章节视频任务 */
  cancelEpisodeVideo(taskId: string) {
    return request.post<{ message: string }>('/batch/cancel-episode-video', { task_id: taskId })
  },

  /** 取消批量任务（一键生图、一键出片） */
  cancelTask(taskId: string) {
    return request.post<{ message: string }>('/batch/cancel-task', { task_id: taskId })
  },
}
