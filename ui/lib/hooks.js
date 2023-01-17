import useSWR, { useSWRConfig } from 'swr'

const INFRA_ADMIN_ROLE = 'admin'

export function useUser() {
  const { cache } = useSWRConfig()
  const { data: user, error, mutate } = useSWR('/api/users/self')
  const { data: org } = useSWR(() => (user ? '/api/organizations/self' : null))
  const { data: { items: grants } = {}, grantsError } = useSWR(() =>
    user
      ? `/api/grants?user=${user?.id}&showInherited=1&resource=infra&limit=1000`
      : null
  )

  return {
    user,
    loading: !user && !error,
    org,
    isAdmin: grants?.some(g => g.privilege === INFRA_ADMIN_ROLE),
    isAdminLoading: !!user && !grants && !grantsError,
    login: async body => {
      const res = await fetch('/api/login', {
        method: 'POST',
        body: JSON.stringify(body),
      })

      const data = await jsonBody(res)

      await mutate()

      return data
    },
    logout: async () => {
      await fetch('/api/logout', { method: 'POST' })

      // clear cache to remove any local user data
      cache.clear()

      await mutate(undefined)
    },
  }
}
