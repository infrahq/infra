import useSWR from 'swr'

const INFRA_ADMIN_ROLE = 'admin'

export function useAdmin() {
  const { data: auth } = useSWR('/api/users/self', { revalidateIfStale: false })
  const { data: { items: grants } = {} } = useSWR(
    `/api/grants?user=${auth?.id}&showInherited=1&resource=infra`,
    { revalidateIfStale: false }
  )

  const loading = !auth && !grants
  const admin = grants?.some(g => g.privilege === INFRA_ADMIN_ROLE)

  return {
    admin,
    loading,
  }
}
