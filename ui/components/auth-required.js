import { useRouter } from 'next/router'
import useSWR from 'swr'

export default function AuthRequired({ children }) {
  console.log('render auth required')
  const { data: auth, error } = useSWR('/api/users/self')
  const router = useRouter()
  const { pathname } = router
  const logout = window.localStorage.getItem('logout')

  if (!auth && !error) {
    return false
  }

  if (!auth?.id) {
    console.log(pathname)
    console.log('logout: ', logout)

    if (!logout && pathname !== '/destinations' && pathname !== '/') {
      router.replace(`/login?next=${pathname.slice(1)}`)
    } else {
      router.replace('/login')
    }

    return false
  }

  return children
}
