import { useRouter } from 'next/router'
import useSWR from 'swr'

export default function AuthRequired({ children }) {
  const { data: auth, error } = useSWR('/api/users/self')
  const router = useRouter()
  const { asPath } = router

  if (!auth && !error) {
    return undefined
  }

  if (!auth?.id) {
    if (asPath !== '/destinations' && asPath !== '/') {
      router.replace(`/login?next=${encodeURIComponent(asPath)}`)
    } else {
      router.replace('/login')
    }
    return undefined
  }
  return children
}
