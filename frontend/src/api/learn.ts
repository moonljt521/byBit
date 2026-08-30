import { http } from './http'
import type { ApiResp } from './types'

export interface LearnItem {
  slug: string
  title: string
}

export interface LearnDoc {
  slug: string
  title: string
  content: string
}

export interface GlossaryTerm {
  term: string
  en: string
  definition: string
}

export async function listCoins() {
  const { data } = await http.get<ApiResp<LearnItem[]>>('/learn/coins')
  return data.data
}

export async function getCoin(slug: string) {
  const { data } = await http.get<ApiResp<LearnDoc>>(`/learn/coins/${slug}`)
  return data.data
}

export async function listConcepts() {
  const { data } = await http.get<ApiResp<LearnItem[]>>('/learn/concepts')
  return data.data
}

export async function getConcept(slug: string) {
  const { data } = await http.get<ApiResp<LearnDoc>>(`/learn/concepts/${slug}`)
  return data.data
}

export async function glossary() {
  const { data } = await http.get<ApiResp<GlossaryTerm[]>>('/learn/glossary')
  return data.data
}
