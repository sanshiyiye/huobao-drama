export interface VideoGeneration {
  id: number
  storyboard_id?: number
  scene_id?: string  // 已废弃，保留用于兼容
  drama_id: string
  image_gen_id?: number
  provider: string
  prompt: string
  model?: string
  image_url?: string
  first_frame_url?: string
  duration?: number
  fps?: number
  resolution?: string
  aspect_ratio?: string
  style?: string
  motion_level?: number
  camera_motion?: string
  seed?: number
  video_url?: string
  local_path?: string
  status: VideoStatus
  task_id?: string
  error_msg?: string
  width?: number
  height?: number
  created_at: string
  updated_at: string
  completed_at?: string
}

export type VideoStatus = 'pending' | 'processing' | 'completed' | 'failed'

export type VideoProvider = 'runway' | 'pika' | 'doubao' | 'openai'

export interface GenerateVideoRequest {
  storyboard_id?: number
  scene_id?: string  // 已废弃，保留用于兼容
  drama_id: string
  image_gen_id?: number
  image_url?: string
  prompt: string
  provider?: string
  model?: string
  duration?: number
  fps?: number
  aspect_ratio?: string
  style?: string
  motion_level?: number
  camera_motion?: string
  seed?: number
  reference_mode?: string   // 参考图模式：single, first_last, multiple, none
  first_frame_url?: string  // 首帧图片URL
  last_frame_url?: string   // 尾帧图片URL
  reference_image_urls?: string[]  // 多图参考模式
}

export interface VideoGenerationListParams {
  drama_id?: string
  storyboard_id?: string
  scene_id?: string  // 已废弃，保留用于兼容
  status?: string  // 支持单个状态或逗号分隔的多个状态，如 "pending,processing"
  page?: number
  page_size?: number
}

export const VIDEO_ASPECT_RATIOS = [
  { label: '16:9 (横屏)', value: '16:9' },
  { label: '9:16 (竖屏)', value: '9:16' },
  { label: '1:1 (正方形)', value: '1:1' },
  { label: '4:3 (传统)', value: '4:3' }
]

export interface VideoMerge {
  id: number
  episode_id: number
  drama_id: number
  title: string
  provider: string
  model?: string
  status: 'pending' | 'processing' | 'completed' | 'failed'
  scenes: any
  merged_url?: string
  duration?: number
  task_id?: string
  error_msg?: string
  created_at: string
  completed_at?: string
  progress?: number
}

export interface SceneClip {
  scene_id: number
  video_url: string
  start_time: number
  end_time: number
  duration: number
  order: number
  transition: any
}

export const CAMERA_MOTIONS = [
  { label: '静止', value: 'static' },
  { label: '推进', value: 'zoom_in' },
  { label: '拉远', value: 'zoom_out' },
  { label: '左移', value: 'pan_left' },
  { label: '右移', value: 'pan_right' },
  { label: '上移', value: 'tilt_up' },
  { label: '下移', value: 'tilt_down' },
  { label: '环绕', value: 'orbit' }
]
