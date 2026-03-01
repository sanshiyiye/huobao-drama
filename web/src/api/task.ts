import request from '../utils/request'

export interface AsyncTask {
    id: string
    type: string
    status: 'pending' | 'processing' | 'completed' | 'failed'
    progress: number
    message: string
    result?: any
    error?: string
    created_at: string
}

export const taskAPI = {
    getStatus(taskId: string) {
        return request.get<AsyncTask>(`/tasks/${taskId}`)
    },
    
    /** 获取资源相关的所有任务 */
    getResourceTasks(resourceId: string) {
        return request.get<AsyncTask[]>(`/tasks?resource_id=${resourceId}`)
    },
    
    /** 删除任务 */
    deleteTask(taskId: string) {
        return request.delete(`/tasks/${taskId}`)
    }
}
