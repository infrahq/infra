import { useRouter } from 'next/router'
import { useEffect, useState } from 'react'
import useSWR from 'swr'
import Head from 'next/head'
import Link from 'next/link'

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

  useEffect(() => {
    setNamespaceResource(`${destination?.name}.${resource}`)
    setRoles(destination?.roles?.filter(r => r != 'cluster-admin'))
  }, [destination, resource])

  const loading = [!adminLoading, users, groups, grants, destination].some(
    x => !x
  )

  return (
    <div className='mb-10'>
      <Head>
        <title>{resource} - Infra</title>
      </Head>
      <header className='mt-6 mb-12 flex items-center justify-between text-xl'>
        <h1 className='flex py-1 font-medium'>
          <Link href='/destinations'>
            <a className='text-gray-500/75 hover:text-gray-600'>Clusters</a>
          </Link>{' '}
          <span className='mx-3 font-light text-gray-400'> / </span>{' '}
          <Link href={`/destinations/${destination?.id}`}>
            <a className='group flex text-gray-500/75 hover:text-gray-600'>
              <div className='mr-2 flex h-8 w-8 flex-none items-center justify-center rounded-md border border-gray-200'>
                <img
                  alt='kubernetes icon'
                  className='h-[18px]'
                  src={`/kubernetes.svg`}
                />
              </div>
              {destination?.name}
            </a>
          </Link>
          <span className='mx-3 font-light text-gray-400'> / </span> {resource}
        </h1>
      </header>
      {!loading && (
        <div className='px-4 sm:px-6 md:px-0'>
          <div className='mt-6 space-y-10'>
            {admin && (
              <div>
                <div className='flex flex-col space-y-2'>
                  <div className='w-full rounded-lg border border-gray-200/75 px-5 py-3'>
                    <h3 className='mb-3 text-base font-medium'>Grant access</h3>
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
