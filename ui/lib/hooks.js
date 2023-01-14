import useSWR, { useSWRConfig } from 'swr'

const INFRA_ADMIN_ROLE = 'admin'

export function useUser() {
  const { cache } = useSWRConfig()
  const { data: user, error } = useSWR('/api/users/self')
  const { data: org } = useSWR(() => (user ? '/api/organizations/self' : null))
  const { data: { items: grants } = {}, grantsError } = useSWR(() =>
    user
      ? `/api/grants?user=${user?.id}&showInherited=1&resource=infra&limit=1000`
      : null
  )

  const loading =
    // User loading
    (!user && !error) ||
    // isAdmin loading
    (!!user && !grants && !grantsError)

  if (loading) {
    return { loading }
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

      for (const key of cache.keys()) {
        cache.delete(key)
      }

      return data
    },

    // logout logs the user out and redirects them to the login page
    logout: async () => {
      await fetch('/api/logout', { method: 'POST' })
      for (const key of cache.keys()) {
        cache.delete(key)
      }
    },
  }
}
