// frontend/src/api/user.js
import request from '@/utils/request'


export function getUserInfo() { return request({ url: '/user/info', method: 'get' }) }