import useSWR from 'swr'

export function useAdmin () {
  const { data: auth } = useSWR('/api/users/self')
  const { data: { items: grants } = {}, error: grantsError } = useSWR(() => `/api/grants?user=${auth.id}&resource=infra`)

  return {
    loading: !grants && !grantsError,
    admin: !!grants?.find(g => g.privilege === 'admin')
  }
}
