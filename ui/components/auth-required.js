import { useRouter } from 'next/router'
import useSWR from 'swr'

export default function AuthRequired({ children }) {
  const { data: auth, error } = useSWR('/api/users/self')
  const router = useRouter()
  const { asPath } = router
  const logout = window.localStorage.getItem('logout')

  if (!auth && !error) {
    return null
  }

  if (!auth?.id) {
    if (!logout && asPath !== '/destinations' && asPath !== '/') {
      router.replace(`/login?next=${encodeURIComponent(asPath)}`)
    } else {
      router.replace('/login')
    }

    return false
  }

  return children
}
