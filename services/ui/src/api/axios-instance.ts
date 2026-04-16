import axios, { type AxiosRequestConfig } from 'axios'

export const axiosInstance = axios.create({
  baseURL: '',
})

// Attach the stored JWT token to every request
axiosInstance.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

export const customInstance = <T>(config: AxiosRequestConfig): Promise<T> => {
  const source = axios.CancelToken.source()
  const promise = axiosInstance({ ...config, cancelToken: source.token }).then(({ data }) => data)
  // @ts-expect-error attach cancel to promise for orval cancellation support
  promise.cancel = () => source.cancel('Query was cancelled')
  return promise
}
