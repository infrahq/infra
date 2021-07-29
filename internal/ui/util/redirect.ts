import { useRouter } from 'next/router'
import { useCookies } from 'react-cookie'

const GRPC_UNAUTHENTICATED_CODE = 16

export function useRedirectToLoginOnUnauthorized(e: any) {
  const router = useRouter()
  const [_, __, removeCookie] = useCookies(['login'])
  if (e && e?.code === GRPC_UNAUTHENTICATED_CODE) {
    removeCookie('login')
    router.replace('/login')
  }
}
