import { useRouter } from 'next/router'
import { useEffect, useState } from 'react'
import useSWR from 'swr'

import { useAdmin } from '../../../lib/admin'

import AccessTable from '../../../components/access-table'
import Dashboard from '../../../components/layouts/dashboard'
import GrantForm from '../../../components/grant-form'

function parent(resource = '') {
  const parts = resource?.split('.')
  return parts?.length > 1 ? parts[0] : null
}

export default function ResourceDetail() {
  const router = useRouter()
  const resource = router.query.resource
  const parentDestinationId = router.query.id

  const [namespaceResource, setNamespaceResource] = useState(null)
  const [roles, setRoles] = useState([])

  const { admin, loading: adminLoading } = useAdmin()

  const { data: destination } = useSWR(
    `/api/destinations/${parentDestinationId}`
  )
  const { data: { items: users } = {} } = useSWR('/api/users?limit=1000')
  const { data: { items: groups } = {} } = useSWR('/api/groups?limit=1000')
  const { data: { items: grants } = {}, mutate } = useSWR(
    `/api/grants?resource=${namespaceResource}&limit=1000`
  )
  const { data: { items: inherited } = {} } = useSWR(() =>
    parent(namespaceResource)
      ? `/api/grants?resource=${parent(namespaceResource)}&limit=1000`
      : null
  )

  useEffect(() => {
    setNamespaceResource(`${destination?.name}.${resource}`)
    setRoles(destination?.roles?.filter(r => r != 'cluster-admin'))
  }, [destination, resource])

  const loading = [!adminLoading, users, groups, grants, destination].some(
    x => !x
  )

  return (
    <div className='md:px-6 xl:px-10 2xl:m-auto 2xl:max-w-6xl'>
      {!loading && (
        <div className='px-4 sm:px-6 md:px-0'>
          <div className='flex min-h-0 flex-1 flex-col px-0 md:px-6 xl:px-0'>
            <div className='py-6 xl:flex xl:items-center xl:justify-between'>
              <div className='min-w-0 flex-1'>
                <div className='flex items-center'>
                  <div>
                    <div className='flex items-center'>
                      <div className='flex flex-row items-center space-x-2'>
                        <h1 className='text-2xl font-bold leading-7 text-gray-900 sm:truncate sm:leading-9'>
                          {resource}
                        </h1>
                        <p className='inline-flex rounded-full bg-blue-100 px-2 text-xs font-semibold leading-5 text-blue-800'>
                          namespace
                        </p>
                      </div>
                    </div>
                    <dl className='mt-6 flex flex-col sm:flex-row sm:flex-wrap'>
                      <dt className='sr-only'>Parent Destination</dt>
                      <dd className='mt-3 flex items-center text-sm font-medium text-gray-500 sm:mr-6 sm:mt-0'>
                        {destination.name}
                      </dd>
                    </dl>
                  </div>
                </div>
              </div>
            </div>
          </div>
          <div className='mt-6 space-y-10'>
            {admin && (
              <div>
                <h2 className='text-md border-b border-gray-200 py-2 font-medium text-gray-500'>
                  Access
                </h2>
                <div className='space-y-6'>
                  <div>
                    <AccessTable
                      grants={grants}
                      users={users}
                      groups={groups}
                      destination={destination}
                      onRemove={async groupId => {
                        await fetch(`/api/grants/${groupId}`, {
                          method: 'DELETE',
                        })
                        mutate({
                          items: grants.filter(x => x.id !== groupId),
                        })
                      }}
                      onChange={async (privilege, group) => {
                        if (privilege === group.privilege) {
                          return
                        }

                        const res = await fetch('/api/grants', {
                          method: 'POST',
                          body: JSON.stringify({
                            ...group,
                            privilege,
                          }),
                        })

                        // delete old grant
                        await fetch(`/api/grants/${group.id}`, {
                          method: 'DELETE',
                        })

                        mutate({
                          items: [
                            ...grants.filter(f => f.id !== group.id),
                            await res.json(),
                          ],
                        })
                      }}
                      inherited={inherited}
                    />
                  </div>
                  <GrantForm
                    roles={roles}
                    onSubmit={async ({ user, group, privilege }) => {
                      // don't add grants that already exist
                      if (
                        grants?.find(
                          g =>
                            g.user === user &&
                            g.group === group &&
                            g.privilege === privilege
                        )
                      ) {
                        return false
                      }

                      const res = await fetch('/api/grants', {
                        method: 'POST',
                        body: JSON.stringify({
                          user,
                          group,
                          privilege,
                          resource: namespaceResource,
                        }),
                      })

                      mutate({ items: [...grants, await res.json()] })
                    }}
                  />
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}

ResourceDetail.layout = page => {
  return <Dashboard>{page}</Dashboard>
}
