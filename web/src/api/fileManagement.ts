import request from '@/utils/request'

// 文件信息类型
export interface FileInfo {
  id: number
  filename: string
  path: string
  url: string
  size: number
  is_associated: boolean
  associated_type?: string
  associated_id?: number
  created_at: string
}

// 列出所有图片文件
export const listImages = async (): Promise<FileInfo[]> => {
  return await request.get('/files/images')
}

// 列出所有视频文件
export const listVideos = async (): Promise<FileInfo[]> => {
  return await request.get('/files/videos')
}

// 删除图片文件
export const deleteImage = async (filename: string): Promise<void> => {
  await request.delete(`/files/images/${filename}`)
}

// 删除视频文件
export const deleteVideo = async (filename: string): Promise<void> => {
  await request.delete(`/files/videos/${filename}`)
}

// 删除文件
export const deleteFile = async (path: string): Promise<void> => {
  await request.delete(`/files/${path}`)
}

// 清理未关联的文件
export const cleanupUnassociatedFiles = async (): Promise<{
  deleted_files: string[]
  count: number
}> => {
  return await request.post('/files/cleanup')
}

// 获取文件大小的格式化字符串
export const formatFileSize = (bytes: number): string => {
  if (bytes === 0) return '0 Bytes'
  const k = 1024
  const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

// 格式化日期
export const formatDate = (dateString: string): string => {
  const date = new Date(dateString)
  return date.toLocaleString()
}
