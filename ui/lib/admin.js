import useSWR from 'swr'

export function useAdmin () {
  const { data: auth } = useSWR('/v1/users/self')
  const { data: grants, error: grantsError } = useSWR(() => `/v1/users/${auth?.id}/grants?resource=infra`)

  return {
    loading: !grants && !grantsError,
    admin: !!grants?.find(g => g.privilege === 'admin')
  }
}
