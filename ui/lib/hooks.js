import { useRouter } from 'next/router'
import useSWR from 'swr'

const INFRA_ADMIN_ROLE = 'admin'

export function useUser({ redirectTo, redirectIfFound } = {}) {
  const { data: user, error } = useSWR('/api/users/self')
  const { data: { items: grants } = {}, grantsError } = useSWR(() =>
    user
      ? `/api/grants?user=${user?.id}&showInherited=1&resource=infra&limit=1000`
      : null
  )
  const router = useRouter()

  const loading =
    // User loading
    (!user && !error) ||
    // isAdmin loading
    (!!user && !grants && !grantsError)

  if (loading) {
    return { loading }
  }

  // Redirect
  if ((redirectTo && !redirectIfFound && !user) || (redirectIfFound && user)) {
    router.replace(redirectTo)
    return { loading: true }
  }

  return {
    user,
    loading,
    isAdmin: grants?.some(g => g.privilege === INFRA_ADMIN_ROLE),
    logout: async () => {
      await fetch('/api/logout', {
        method: 'POST',
      })
      window.location = '/login'
    },
  }
}
