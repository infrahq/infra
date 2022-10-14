import { useRouter } from 'next/router'
import useSWR from 'swr'

const INFRA_ADMIN_ROLE = 'admin'

export function useUser({ redirectTo, redirectIfFound } = {}) {
  const { data: user, error, isValidating, mutate } = useSWR('/api/users/self')
  const { data: org } = useSWR('/api/organizations/self')
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

  // Redirect only if we aren't loading or fetching new data
  if (
    !loading &&
    !isValidating &&
    ((redirectTo && !redirectIfFound && !user) || (redirectIfFound && user))
  ) {
    if (router.asPath === "/device")
      router.replace(`/login/organizations?next=${encodeURIComponent(router.asPath)}`)
    else 
      router.replace(redirectTo)
    return { loading: true }
  }

  return {
    user,
    loading,
    org,
    isAdmin: grants?.some(g => g.privilege === INFRA_ADMIN_ROLE),

    // login logs the user in and clears the local cache
    login: async body => {
      const res = await fetch('/api/login', {
        method: 'POST',
        body: JSON.stringify(body),
      })

      const data = await jsonBody(res)

      // Don't reset state if a password change is required
      if (data.passwordUpdateRequired) {
        return data
      }

      // Clear user cache
      mutate(undefined, false)

      return data
    },

    // logout logs the user out and redirects them to the login page
    logout: async () => {
      await fetch('/api/logout', {
        method: 'POST',
      })

      // Do a "hard" redirect in order to clear the swr cache of all data
      window.location = '/login'
    },
  }
}
