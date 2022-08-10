import useSWR from 'swr'
import { useState } from 'react'
import Head from 'next/head'

import { sortBySubject } from '../../lib/grants'

import GrantForm from '../../components/grant-form'
import Dashboard from '../../components/layouts/dashboard'
import DeleteModal from '../../components/delete-modal'

function AdminList({ grants, users, groups, onRemove, auth, selfGroups }) {
  const [open, setOpen] = useState(false)
  const [deleteId, setDeleteId] = useState(null)

  const grantsList = grants?.sort(sortBySubject)?.map(grant => {
    const message =
      grant?.user === auth?.id
        ? 'Are you sure you want to revoke your own admin access?'
        : selfGroups?.some(g => g.id === grant.group)
        ? `Are you sure you want to revoke this group's admin access? You are a member of this group.`
        : undefined

    const name =
      users?.find(u => grant.user === u.id)?.name ||
      groups?.find(group => grant.group === group.id)?.name ||
      ''

    return { ...grant, message, name }
  })

  return (
    <div className='-my-2 -mx-4 overflow-x-auto sm:-mx-6 lg:-mx-8'>
      <div className='inline-block min-w-full py-2 px-4 align-middle md:px-6 lg:px-8'>
        <div className='overflow-hidden shadow ring-1 ring-black ring-opacity-5 md:rounded-lg'>
          <table className='min-w-full divide-y divide-gray-300'>
            <thead className='bg-gray-50'>
              <tr>
                <th
                  scope='col'
                  className='py-3.5 pl-4 pr-3 text-left text-xs font-semibold text-gray-900 sm:pl-6'
                >
                  Infra Admin
                </th>
              </tr>
            </thead>
            <tbody className='bg-white'>
              {grantsList?.map(grant => (
                <tr key={grant.id}>
                  <td className='whitespace-nowrap py-4 pl-4 pr-3 text-xs font-medium sm:pl-6'>
                    <div>{grant.name}</div>
                    <div>
                      <button
                        onClick={() => {
                          setDeleteId(grant.id)
                          setOpen(true)
                        }}
                        className='cursor-pointer text-4xs uppercase text-gray-800 hover:text-gray-400'
                      >
                        Revoke
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          <DeleteModal
            open={open}
            setOpen={setOpen}
            primaryButtonText='Revoke'
            onSubmit={() => {
              onRemove(deleteId)
              setOpen(false)
            }}
            title='Revoke Admin'
            message={
              !grantsList?.find(grant => grant.id === deleteId)?.message ? (
                <>
                  Are you sure you want to revoke admin access for{' '}
                  <span className='font-bold'>
                    {grantsList?.find(grant => grant.id === deleteId)?.name}
                  </span>
                  ?
                </>
              ) : (
                grantsList?.find(grant => grant.id === deleteId)?.message
              )
            }
          />
        </div>
      </div>
    </div>
  )
}

export default function Settings() {
  const { data: auth } = useSWR('/api/users/self')

  const { data: { items: users } = {} } = useSWR('/api/users?limit=1000')
  const { data: { items: groups } = {} } = useSWR('/api/groups?limit=1000')
  const { data: { items: grants } = {}, mutate } = useSWR(
    '/api/grants?resource=infra&privilege=admin&limit=1000'
  )
  const { data: { items: selfGroups } = {} } = useSWR(
    `/api/groups?userID=${auth?.id}&limit=1000`
  )

  return (
    <div className='my-10 px-6 xl:px-10 2xl:mx-auto 2xl:max-w-6xl'>
      <Head>
        <title>Settings - Infra</title>
      </Head>
      <div className='flex flex-1 flex-col'>
        <h1 className='text-lg font-medium'>Infra Administrators</h1>
        <p className='mt-1 text-sm text-gray-500'>
          These users have full access to Infra and all infrastructure.
        </p>
        <div className='flex flex-col space-y-3 pt-6'>
          <GrantForm
            resource='infra'
            roles={['admin']}
            onSubmit={async ({ user, group }) => {
              // don't add grants that already exist
              if (grants?.find(g => g.user === user && g.group === group)) {
                return false
              }

              const res = await fetch('/api/grants', {
                method: 'POST',
                body: JSON.stringify({
                  user,
                  group,
                  privilege: 'admin',
                  resource: 'infra',
                }),
              })

              mutate({ items: [...grants, await res.json()] })
            }}
          />
          <div>
            <AdminList
              grants={grants}
              users={users}
              groups={groups}
              selfGroups={selfGroups}
              auth={auth}
              onRemove={async grantId => {
                await fetch(`/api/grants/${grantId}`, {
                  method: 'DELETE',
                })
                mutate({ items: grants?.filter(x => x.id !== grantId) })
              }}
            />
          </div>
        </div>
      </div>
    </div>
  )
}
Settings.layout = page => {
  return <Dashboard>{page}</Dashboard>
}
