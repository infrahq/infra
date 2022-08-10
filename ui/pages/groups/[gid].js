import { useRouter } from 'next/router'
import useSWR from 'swr'
import { useState, useRef } from 'react'
import { PlusIcon, UserIcon } from '@heroicons/react/outline'

import { useAdmin } from '../../lib/admin'

import EmptyData from '../../components/empty-data'
import RemoveButton from '../../components/remove-button'
import GrantsList from '../../components/grants-list'
import DeleteModal from '../../components/delete-modal'
import TypeaheadCombobox from '../../components/typeahead-combobox'
import Dashboard from '../../components/layouts/dashboard'

function EmailsSelectInput({
  selectedEmails,
  setSelectedEmails,
  existMembers,
  onClick,
}) {
  const { data: { items: users } = { items: [] } } = useSWR(
    '/api/users?limit=1000'
  )

  const [query, setQuery] = useState('')
  const inputRef = useRef(null)

  const selectedEmailsId = selectedEmails.map(i => i.id)

  const filteredEmail = [...users.map(u => ({ ...u, user: true }))]
    .filter(s => s?.name?.toLowerCase()?.includes(query.toLowerCase()))
    .filter(s => !selectedEmailsId?.includes(s.id))
    .filter(s => !existMembers?.includes(s.id))

  const removeSelectedEmail = email => {
    setSelectedEmails(selectedEmails.filter(item => item.id !== email.id))
  }

  return (
    <section className='flex'>
      <div className='flex flex-1 items-center rounded-md bg-gray-100'>
        <TypeaheadCombobox
          selectedEmails={selectedEmails}
          setSelectedEmails={setSelectedEmails}
          onRemove={removedEmail => removeSelectedEmail(removedEmail)}
          inputRef={inputRef}
          setQuery={setQuery}
          filteredEmail={filteredEmail}
          onKeyDownEvent={key => {
            if (key === 'Backspace' && inputRef.current.value.length === 0) {
              removeSelectedEmail(selectedEmails[selectedEmails.length - 1])
            }
          }}
        />
      </div>
      <div className='p-3'>
        <button
          type='button'
          onClick={onClick}
          disabled={selectedEmails.length === 0}
          className='inline-flex items-center rounded-md border border-transparent bg-black px-4 py-2 text-2xs font-medium text-white shadow-sm hover:bg-gray-800'
        >
          <PlusIcon className='mr-1.5 h-3 w-3' />
          <div className='text-violet-100'>Add</div>
        </button>
      </div>
    </section>
  )
}

function UsersInGroupTable({ users, authId, onRemove = () => {} }) {
  const [open, setOpen] = useState(false)

  return (
    <table className='min-w-full divide-y divide-gray-300'>
      <tbody className='bg-white'>
        {users
          .sort((a, b) => a.id?.localeCompare(b.id))
          .map(user => (
            <tr key={user.id} className='border-b border-gray-200'>
              <td className='whitespace-nowrap py-4 text-xs font-medium'>
                <div className='font-medium text-gray-900'>{user.name}</div>
              </td>
              <td className='py-4 px-3 text-right text-sm text-gray-500'>
                <button
                  onClick={() => {
                    if (user.id === authId) {
                      setOpen(true)
                      return
                    }

                    onRemove(user.id)
                  }}
                  className='text-xs text-blue-600 hover:text-blue-900'
                >
                  Remove
                </button>
                <DeleteModal
                  open={open}
                  setOpen={setOpen}
                  primaryButtonText='Remove'
                  onSubmit={() => {
                    onRemove(user.id)
                    setOpen(false)
                  }}
                  title='Remove User'
                  message='Are you sure you want to remove yourself from this group? You will lose any access provided by this group.'
                />
              </td>
            </tr>
          ))}
      </tbody>
    </table>
  )
}

export default function GroupDetail() {
  const router = useRouter()
  const groupId = router.query.gid

  const { data: group, mutate: mutateGroups } = useSWR(`/api/groups/${groupId}`)
  const { admin, loading: adminLoading } = useAdmin()

  const { data: auth } = useSWR('/api/users/self')
  const { data: { items: users } = {}, mutate: mutateUsers } = useSWR(
    `/api/users?group=${group?.id}&limit=1000`
  )
  const { data: { items } = {}, mutate: mutateGrants } = useSWR(
    `/api/grants?group=${group?.id}&limit=1000`
  )
  const { data: { items: infraAdmins } = {} } = useSWR(
    '/api/grants?resource=infra&privilege=admin&limit=1000'
  )

  const [emails, setEmails] = useState([])

  const grants = items?.filter(g => g.resource !== 'infra')
  const existMembers = users?.map(m => m.id)
  const adminGroups = infraAdmins?.map(admin => admin.group)

  const loading = [!adminLoading, auth, users, grants, infraAdmins].some(
    x => !x
  )

  const hideRemoveGroupBtn =
    !admin || (infraAdmins?.length === 1 && adminGroups.includes(group?.id))

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
                        {group?.name}
                      </h1>
                    </div>
                    <dl className='mt-6 flex flex-col sm:flex-row sm:flex-wrap'>
                      <dt className='sr-only'>Group Id</dt>
                      <dd className='flex items-center text-sm font-medium text-gray-500 sm:mr-6'>
                        {group?.id}
                      </dd>
                      <dt className='sr-only'>Number of users</dt>
                      <dd className='mt-3 flex items-center text-sm font-medium text-gray-500 sm:mr-6 sm:mt-0'>
                        <UserIcon
                          className='mr-1.5 h-5 w-5 flex-shrink-0 text-gray-400'
                          aria-hidden='true'
                        />
                        {group?.totalUsers}{' '}
                        {group?.totalUsers === 1 ? 'member' : 'members'}
                      </dd>
                    </dl>
                  </div>
                </div>
              </div>
              <div className='mt-6 flex space-x-3 xl:mt-0 xl:ml-4'>
                {!hideRemoveGroupBtn && (
                  <RemoveButton
                    onRemove={async () => {
                      await fetch(`/api/groups/${groupId}`, {
                        method: 'DELETE',
                      })

                      router.replace('/groups')
                    }}
                    modalTitle='Remove group'
                    modalMessage={
                      <>
                        Are you sure you want to delete{' '}
                        <span className='font-bold'>{group?.name}</span>? This
                        action cannot be undone.
                      </>
                    }
                  >
                    Remove this group
                  </RemoveButton>
                )}
              </div>
            </div>
          </div>
          <div className='mt-6 space-y-10'>
            <div>
              <h2 className='text-md border-b border-gray-200 py-2 font-medium text-gray-500'>
                Users
              </h2>
              <div className='space-y-4'>
                <div>
                  {users?.length > 0 && (
                    <UsersInGroupTable
                      users={users}
                      authId={auth.id}
                      onRemove={async userId => {
                        await fetch(`/api/groups/${group?.id}/users`, {
                          method: 'PATCH',
                          body: JSON.stringify({
                            usersToRemove: [userId],
                          }),
                        })

                        const filteredUsers = users.filter(i => i.id !== userId)
                        mutateUsers({
                          items: filteredUsers,
                        })
                        mutateGroups({
                          totalUsers: filteredUsers.length,
                        })
                      }}
                    />
                  )}
                </div>
                <div className='flex flex-col'>
                  <EmailsSelectInput
                    selectedEmails={emails}
                    setSelectedEmails={setEmails}
                    existMembers={existMembers}
                    onClick={async () => {
                      const usersToAdd = emails.map(email => email.id)
                      await fetch(`/api/groups/${group?.id}/users`, {
                        method: 'PATCH',
                        body: JSON.stringify({ usersToAdd }),
                      })

                      const totalUsers = users.length + emails.length

                      mutateUsers({ items: [...users, ...emails] })
                      mutateGroups({ totalUsers: totalUsers })
                      setEmails([])
                    }}
                  />
                </div>
              </div>
            </div>
            <div>
              <h2 className='text-md border-b border-gray-200 py-2 font-medium text-gray-500'>
                Access
              </h2>
              <div className='space-y-6'>
                <div>
                  <GrantsList
                    grants={grants}
                    onRemove={async id => {
                      await fetch(`/api/grants/${id}`, { method: 'DELETE' })

                      const items = grants.filter(x => x.id !== id)
                      mutateGrants({ items })
                      mutateGroups({ totalUsers: items.length })
                    }}
                    onChange={async (privilege, grant) => {
                      const res = await fetch('/api/grants', {
                        method: 'POST',
                        body: JSON.stringify({
                          ...grant,
                          privilege,
                        }),
                      })

                      // delete old grant
                      await fetch(`/api/grants/${grant.id}`, {
                        method: 'DELETE',
                      })
                      mutateGrants({
                        items: [
                          ...grants.filter(f => f.id !== grant.id),
                          await res.json(),
                        ],
                      })
                    }}
                  />
                </div>
              </div>
              {!grants?.length && (
                <EmptyData>
                  <div className='mt-6'>No access</div>
                </EmptyData>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

GroupDetail.layout = page => {
  return <Dashboard>{page}</Dashboard>
}
