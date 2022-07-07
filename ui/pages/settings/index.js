import Head from 'next/head'
import Link from 'next/link'
import { useRouter } from 'next/router'
import { useState } from 'react'
import useSWR from 'swr'

import { sortBySubject } from '../../lib/grants'
import { useAdmin } from '../../lib/admin'

import Dashboard from '../../components/layouts/dashboard'
import DeleteModal from '../../components/delete-modal'
import Notification from '../../components/notification'
import GrantForm from '../../components/grant-form'

function AdminGrant({ name, showRemove, onRemove }) {
  const [open, setOpen] = useState(false)

  return (
    <div className='group flex items-center justify-between py-1 text-2xs'>
      <div className='py-1.5'>{name}</div>
      {showRemove && (
        <div className='flex justify-end text-right opacity-0 group-hover:opacity-100'>
          <button
            onClick={() => setOpen(true)}
            className='-mr-2 flex-none cursor-pointer px-2 py-1 text-2xs text-gray-500 hover:text-violet-100'
          >
            Revoke
          </button>
          <DeleteModal
            open={open}
            setOpen={setOpen}
            primaryButtonText='Revoke'
            onSubmit={onRemove}
            title='Revoke Admin'
            message={
              <>
                Are you sure you want to revoke admin access for{' '}
                <span className='font-bold text-white'>{name}</span>?
              </>
            }
          />
        </div>
      )}
    </div>
  )
}

export default function Settings() {
  const router = useRouter()
  const { data: auth } = useSWR('/api/users/self')
  const { admin } = useAdmin()

  const { resetPassword } = router.query
  const [showNotification, setshowNotification] = useState(
    resetPassword === 'success'
  )

  const { data: { items: users } = {} } = useSWR('/api/users')
  const { data: { items: groups } = {} } = useSWR('/api/groups')
  const { data: { items: grants } = {}, mutate } = useSWR(
    '/api/grants?resource=infra&privilege=admin'
  )

  const hasInfraProvider = auth?.providerNames.includes('infra')

  return (
    <>
      <Head>
        <title>Settings - Infra</title>
      </Head>
      {auth && (
        <div className='mt-6 mb-4 flex flex-1 flex-col space-y-8'>
          <h1 className='mb-6 text-xs font-bold'>Settings</h1>
          {hasInfraProvider && (
            <div className='w-full max-w-md pb-12'>
              <div className='border-b border-gray-800 pb-3 text-2xs uppercase leading-none text-gray-400'>
                Account
              </div>
              <div className='flex flex-col space-y-2 pt-6'>
                <div className='group flex'>
                  <div className='flex flex-1 items-center'>
                    <div className='w-[26%] text-2xs text-gray-400'>Email</div>
                    <div className='text-2xs'>{auth?.name}</div>
                  </div>
                </div>
                <div className='group flex'>
                  <div className='flex flex-1 items-center'>
                    <div className='w-[30%] text-2xs text-gray-400'>
                      Password
                    </div>
                    <div className='text-2xs'>********</div>
                  </div>
                  <div className='flex justify-end'>
                    <Link href='/settings/password-reset'>
                      <a className='-mr-2 flex-none cursor-pointer p-2 text-2xs uppercase text-gray-500 hover:text-violet-100'>
                        Change
                      </a>
                    </Link>
                  </div>
                </div>
              </div>
            </div>
          )}
          {resetPassword && (
            <Notification
              show={showNotification}
              setShow={setshowNotification}
              text='Password Successfully Reset'
            />
          )}
        </div>
      )}
      {admin && (
        <div className='max-w-md'>
          <div className='border-b border-gray-800 pb-6 text-2xs uppercase leading-none text-gray-400'>
            Admins
          </div>
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
          <div className='mt-6'>
            {grants?.sort(sortBySubject)?.map(g => (
              <AdminGrant
                key={g.id}
                name={
                  users?.find(u => g.user === u.id)?.name ||
                  groups?.find(group => g.group === group.id)?.name ||
                  ''
                }
                showRemove={g?.user !== auth?.id}
                onRemove={async () => {
                  await fetch(`/api/grants/${g.id}`, { method: 'DELETE' })
                  mutate({ items: grants.filter(x => x.id !== g.id) })
                }}
              />
            ))}
          </div>
        </div>
      )}
    </>
  )
}

Settings.layout = function (page) {
  return <Dashboard>{page}</Dashboard>
}
