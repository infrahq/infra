import { useRouter } from 'next/router'
import { useCookies } from 'react-cookie'

export function useRedirectToLoginOnUnauthorized(e: any) {
  const router = useRouter()
  const [_, __, removeCookie] = useCookies(['login'])
  if (e && e?.status === 401) {
    removeCookie('login')
    router.replace('/login')
  }
}
