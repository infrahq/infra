import { ViewGridIcon } from '@heroicons/react/outline'
import { useRouter } from 'next/router'
import { useState } from 'react'
import useSWR from 'swr'

import { useAdmin } from '../../lib/admin'
import { sortByResource } from '../../lib/grants'

import DeleteModal from '../../components/delete-modal'
import EmptyData from '../../components/empty-data'
import Dashboard from '../../components/layouts/dashboard'
import RemoveButton from '../../components/remove-button'
import RoleSelect from '../../components/role-select'
import Tooltip from '../../components/tooltip'

function UserGroupsTable({
  groups,
  authId,
  userId,
  adminGroups,
  onRemove = () => {},
}) {
  const [open, setOpen] = useState(false)

  return (
    <table className='min-w-full divide-y divide-gray-300'>
      <tbody className='bg-white'>
        {groups.map(group => {
          return (
            <tr key={group.id} className='border-b border-gray-200'>
              <td className='whitespace-nowrap py-4 text-xs font-medium'>
                <div className='font-medium text-gray-900'>{group.name}</div>
              </td>
              <td className='py-4 px-3 text-right text-sm text-gray-500'>
                <button
                  onClick={() => {
                    if (authId === userId && adminGroups.includes(group.id)) {
                      setOpen(true)
                      return
                    }
                    onRemove(group.id)
                  }}
                  className='text-xs text-blue-600 hover:text-blue-900'
                >
                  Remove
                </button>
                <DeleteModal
                  open={open}
                  setOpen={setOpen}
                  primaryButtonText='Remove'
                  onSubmit={() => onRemove(group.id)}
                  title='Remove Group'
                  message='Are you sure you want to remove yourself from this group? You will lose any access that this group grants.'
                />
              </td>
            </tr>
          )
        })}
      </tbody>
    </table>
  )
}

export default function UserDetail() {
  const router = useRouter()
  const userId = router.query.uid

  const { data: user } = useSWR(`/api/users/${userId}`)
  const { data: auth } = useSWR('/api/users/self')

  const { data: { items } = {}, mutate } = useSWR(
    `/api/grants?user=${user?.id}&showInherited=1&limit=1000`
  )
  const { data: { items: groups } = {}, mutate: mutateGroups } = useSWR(
    `/api/groups?userID=${user?.id}&limit=1000`
  )

  const { data: { items: infraAdmins } = {} } = useSWR(
    '/api/grants?resource=infra&privilege=admin&limit=1000'
  )

  const { admin, loading: adminLoading } = useAdmin()

  const grants = items?.filter(g => g.resource !== 'infra')
  const adminGroups = infraAdmins?.map(admin => admin.group)

  const loading = [!adminLoading, auth, grants, groups, user].some(x => !x)

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
                      <h1 className='text-2xl font-bold leading-7 text-gray-900 sm:truncate sm:leading-9'>
                        {user?.name}
                      </h1>
                    </div>
                    <dl className='mt-6 flex flex-col sm:flex-row sm:flex-wrap'>
                      <dt className='sr-only'>Providers</dt>
                      <dd className='mt-3 flex items-center text-sm font-medium text-gray-500 sm:mr-6 sm:mt-0'>
                        <ViewGridIcon
                          className='mr-1.5 h-5 w-5 flex-shrink-0 text-gray-400'
                          aria-hidden='true'
                        />
                        {user?.providerNames.join(', ')}
                      </dd>
                    </dl>
                  </div>
                </div>
              </div>
              <div className='mt-6 flex space-x-3 xl:mt-0 xl:ml-4'>
                {auth.id !== user?.id && (
                  <RemoveButton
                    onRemove={async () => {
                      await fetch(`/api/users/${userId}`, {
                        method: 'DELETE',
                      })

                      router.replace('/users')
                    }}
                    modalTitle='Remove User'
                    modalMessage={
                      <>
                        Are you sure you want to remove{' '}
                        <span className='font-bold'>{user?.name}?</span>
                      </>
                    }
                  >
                    Remove user
                  </RemoveButton>
                )}
              </div>
            </div>
          </div>
          {admin && (
            <div className='mt-6 space-y-10'>
              <div>
                <h2 className='text-md border-b border-gray-200 py-2 font-medium text-gray-500'>
                  Groups
                </h2>
                <div>
                  {groups?.length > 0 ? (
                    <UserGroupsTable
                      groups={groups}
                      authId={auth.id}
                      userId={userId}
                      adminGroups={adminGroups}
                      onRemove={async groupId => {
                        const usersToRemove = [user.id]
                        await fetch(`/api/groups/${groupId}/users`, {
                          method: 'PATCH',
                          body: JSON.stringify({ usersToRemove }),
                        })
                        mutateGroups({
                          items: groups.filter(i => i.id !== groupId),
                        })
                      }}
                    />
                  ) : (
                    <EmptyData>
                      <div className='mt-6'>No groups</div>
                    </EmptyData>
                  )}
                </div>
              </div>
              <div>
                <h2 className='text-md border-b border-gray-200 py-2 font-medium text-gray-500'>
                  Access
                </h2>
                <div className='space-y-6'>
                  <div>
                    <table className='min-w-full divide-y divide-gray-300'>
                      <tbody className='bg-white'>
                        {grants
                          ?.sort(sortByResource)
                          ?.sort((a, b) => {
                            if (a.user === user.id) {
                              return -1
                            }

                            if (b.user === user.id) {
                              return 1
                            }

                            return 0
                          })
                          .map(g => (
                            <tr key={g.id} className='border-b border-gray-200'>
                              <td className='whitespace-nowrap py-4'>
                                <div className='truncate text-sm font-medium text-gray-900'>
                                  {g.resource}
                                </div>
                              </td>
                              <td className='py-4 px-3'>
                                {g.user !== user.id ? (
                                  <div className='flex items-center justify-end space-x-6'>
                                    <Tooltip
                                      message='This access is inherited by a group and
                                        cannot be edited here'
                                      direction='left'
                                    >
                                      <p className='flex rounded-full bg-gray-900 px-2 text-xs font-semibold leading-5 text-gray-200'>
                                        inherited
                                      </p>
                                    </Tooltip>
                                    <div className='relative py-2 text-sm text-gray-700'>
                                      {g.privilege}
                                    </div>
                                  </div>
                                ) : (
                                  <RoleSelect
                                    role={g.privilege}
                                    resource={g.resource}
                                    remove
                                    direction='left'
                                    onRemove={async () => {
                                      await fetch(`/api/grants/${g.id}`, {
                                        method: 'DELETE',
                                      })
                                      mutate({
                                        items: grants.filter(
                                          x => x.id !== g.id
                                        ),
                                      })
                                    }}
                                    onChange={async privilege => {
                                      const res = await fetch('/api/grants', {
                                        method: 'POST',
                                        body: JSON.stringify({
                                          ...g,
                                          privilege,
                                        }),
                                      })

                                      // delete old grant
                                      await fetch(`/api/grants/${g.id}`, {
                                        method: 'DELETE',
                                      })
                                      mutate({
                                        items: [
                                          ...grants.filter(f => f.id !== g.id),
                                          await res.json(),
                                        ],
                                      })
                                    }}
                                  />
                                )}
                              </td>
                            </tr>
                          ))}
                      </tbody>
                    </table>
                  </div>
                </div>
                <div>
                  {!grants?.length && !loading && (
                    <EmptyData>
                      <div className='mt-6'>No access</div>
                    </EmptyData>
                  )}
                </div>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

UserDetail.layout = page => {
  return <Dashboard>{page}</Dashboard>
}
