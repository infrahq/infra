import { useRouter } from 'next/router'
import useSWR from 'swr'

export default function AuthRequired({ children }) {
  const { data: auth, error } = useSWR('/api/users/self')
  const router = useRouter()
  const { pathname } = router
  const logout = window.localStorage.getItem('logout')

  if (!auth && !error) {
    return null
  }

  if (!auth?.id) {
    if (!logout && pathname !== '/destinations' && pathname !== '/') {
      router.replace(`/login?next=${pathname.slice(1)}`)
    } else {
      router.replace('/login')
    }

    return false
  }

  return children
}
