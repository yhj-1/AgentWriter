/**
 * 日期工具函数
 */
import dayjs from 'dayjs'

/**
 * 格式化日期
 * @param date 日期字符串或时间戳
 * @param format 格式化模板，默认 'YYYY-MM-DD HH:mm'
 */
export const formatDate = (date: string | number, format = 'YYYY-MM-DD HH:mm'): string => {
  return dayjs(date).format(format)
}

/**
 * 格式化日期（短格式）
 * @param date 日期字符串或时间戳
 */
export const formatDateShort = (date: string | number): string => {
  return formatDate(date, 'MM-DD HH:mm')
}

/**
 * 格式化日期（完整格式，含秒）
 * @param date 日期字符串或时间戳
 */
export const formatDateFull = (date: string | number): string => {
  return formatDate(date, 'YYYY-MM-DD HH:mm:ss')
}
