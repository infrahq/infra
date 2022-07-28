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
      router.replace(`/login?next=${asPath.slice(1)}`)
    } else {
      router.replace('/login')
    }

    return false
  }

  return children
}
