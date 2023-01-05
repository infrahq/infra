import React, { useState, useEffect, useRef } from 'react'
import Head from 'next/head'
import { useRouter } from 'next/router'
import Link from 'next/link'
import useSWR, { mutate } from 'swr'
import moment from 'moment'
import {
  TrashIcon,
  ChevronDownIcon,
  CheckIcon,
  PlusIcon,
} from '@heroicons/react/24/outline'
import { Menu } from '@headlessui/react'

import { useUser } from '../../lib/hooks'
import { sortBySubject } from '../../lib/grants'
import { formatPasswordRequirements } from '../../lib/login'
import Notification from '../../components/notification'
import GrantForm from '../../components/grant-form'
import Dashboard from '../../components/layouts/dashboard'
import DeleteModal from '../../components/delete-modal'
import Table from '../../components/table'
import { googleSocialLoginID } from '../../lib/providers'

const CREATE_ACCESS_KEY_SCOPE = 'create-key'
const CONNECTOR_USER = 'connector'

function PersonalKeys() {
  const { user } = useUser()
  const { data: { items: keys } = {}, mutate: mutate } = useSWR(() =>
    user ? `/api/access-keys?limit=1000&userID=${user.id}` : null
  )

  return (
    <>
      <header className='my-6 flex flex-col justify-between space-y-4 md:flex-row md:space-y-0 md:space-x-4'>
        <div className='flex-1'>
          <h2 className='mb-0.5 font-display text-lg font-medium'>
            Personal Keys
          </h2>
          <h3 className='text-sm text-gray-500'>
            Personal keys are used to authenticate with Infra using the API or
            CLI. These keys share the same permissions as your user.
          </h3>
        </div>
        <Link
          href='/settings/access-key/add'
          className='ml-4 inline-flex items-center self-end rounded-md border border-transparent bg-black px-4 py-2 text-xs font-medium text-white shadow-sm hover:cursor-pointer hover:bg-gray-800'
        >
          <PlusIcon className='mr-1 h-3 w-3' /> Personal Key
        </Link>
      </header>
      <div className='mt-3 flex min-h-0 flex-1 flex-col'>
        <Table
          data={keys
            // Hide connector keys
            ?.filter(k => k.issuedForName !== CONNECTOR_USER)
            // Hide login session keys
            .filter(k => !k.scopes?.includes(CREATE_ACCESS_KEY_SCOPE))}
          empty='No personal keys'
          columns={[
            {
              cell: function Cell(info) {
                return (
                  <div className='flex flex-col py-0.5'>
                    <div className='truncate text-sm font-medium text-gray-700'>
                      {info.getValue()}
                    </div>
                    {info.row.original.created && (
                      <div className='space-y-1 pt-2 text-3xs text-gray-500 sm:hidden'>
                        created -{' '}
                        <span className='font-semibold text-gray-700'>
                          {moment(info.row.original.created).from()}
                        </span>
                      </div>
                    )}
                    {info.row.original.lastUsed && (
                      <div className='space-y-1 pt-2 text-3xs text-gray-500 sm:hidden'>
                        last used -{' '}
                        <span className='font-semibold text-gray-700'>
                          {moment(info.row.original.lastUsed).from()}
                        </span>
                      </div>
                    )}
                    <div className='space-y-1 pt-2 text-3xs text-gray-500 sm:hidden'>
                      the key will expire on{' '}
                      <span className='font-semibold text-gray-700'>
                        {moment(info.row.original.expires).format('YYYY/MM/DD')}
                      </span>
                    </div>
                  </div>
                )
              },
              header: () => <span>Name</span>,
              accessorKey: 'name',
            },
            {
              cell: info => (
                <div className='hidden sm:table-cell'>
                  {info.getValue() ? moment(info.getValue()).from() : '-'}
                </div>
              ),
              header: () => (
                <span className='hidden sm:table-cell'>Created</span>
              ),
              accessorKey: 'created',
            },
            {
              cell: info => (
                <div className='hidden sm:table-cell'>
                  {info.getValue() ? moment(info.getValue()).from() : '-'}
                </div>
              ),
              header: () => (
                <span className='hidden sm:table-cell'>Last used</span>
              ),
              accessorKey: 'lastUsed',
            },
            {
              cell: info => (
                <div className='hidden sm:table-cell'>
                  {info.getValue() ? moment(info.getValue()).from() : '-'}
                </div>
              ),
              header: () => (
                <span className='hidden sm:table-cell'>Expires</span>
              ),
              accessorKey: 'expires',
            },
            {
              id: 'delete',
              cell: function Cell(info) {
                const [openDeleteModal, setOpenDeleteModal] = useState(false)

                const { name, id } = info.row.original

                return (
                  <div className='flex justify-end'>
                    <button
                      type='button'
                      onClick={() => {
                        setOpenDeleteModal(true)
                      }}
                      className='group flex w-full items-center rounded-md bg-white px-2 py-1.5 text-xs font-medium text-red-500'
                    >
                      <TrashIcon className='mr-2 h-3.5 w-3.5' />
                      <span className='hidden sm:block'>Remove</span>
                    </button>
                    <DeleteModal
                      open={openDeleteModal}
                      setOpen={setOpenDeleteModal}
                      primaryButtonText='Remove'
                      onSubmit={async () => {
                        try {
                          await fetch(`/api/access-keys/${id}`, {
                            method: 'DELETE',
                          })
                        } catch (e) {
                          console.log(e)
                        }

                        setOpenDeleteModal(false)
                        mutate()
                      }}
                      title='Remove Access Key'
                      message={
                        <div>
                          Are you sure you want to remove access key:{' '}
                          <span className='break-all font-bold'>{name}</span>?
                        </div>
                      }
                    />
                  </div>
                )
              },
            },
          ]}
        />
      </div>
    </>
  )
}

function ConnectorKeys() {
  const { data: { items: connectors } = {} } = useSWR(
    '/api/users?name=connector&showSystem=true'
  )
  const { data: { items: accessKeys } = {}, mutate: mutate } = useSWR(() =>
    connectors?.[0]?.id
      ? `/api/access-keys?userID=${connectors?.[0]?.id}&limit=1000`
      : null
  )

  return (
    <>
      <header className='my-6 flex flex-col justify-between space-y-4 md:flex-row md:space-y-0 md:space-x-4'>
        <div>
          <h2 className='mb-0.5 flex items-center font-display text-lg font-medium'>
            Connector Keys
          </h2>
          <h3 className='text-sm text-gray-500'>
            Connector keys are used to connect infrastructure to Infra and have
            limited permissions. These keys are shared by your organization.
          </h3>
        </div>
        <Link
          href='/settings/access-key/add?connector=true'
          className='inline-flex items-center self-end whitespace-nowrap rounded-md border border-transparent bg-black px-3 py-2 text-xs font-medium text-white shadow-sm hover:cursor-pointer hover:bg-gray-800'
        >
          <PlusIcon className='mr-1 h-3 w-3' /> Connector Key
        </Link>
      </header>
      <div className='mt-3 flex min-h-0 flex-1 flex-col'>
        <Table
          data={accessKeys}
          empty='No connector keys'
          columns={[
            {
              cell: function Cell(info) {
                return (
                  <div className='flex flex-col py-0.5'>
                    <div className='truncate text-sm font-medium text-gray-700'>
                      {info.getValue()}
                    </div>
                    {info.row.original.created && (
                      <div className='space-y-1 pt-2 text-3xs text-gray-500 sm:hidden'>
                        created -{' '}
                        <span className='font-semibold text-gray-700'>
                          {moment(info.row.original.created).from()}
                        </span>
                      </div>
                    )}
                    {info.row.original.lastUsed && (
                      <div className='space-y-1 pt-2 text-3xs text-gray-500 sm:hidden'>
                        last used -{' '}
                        <span className='font-semibold text-gray-700'>
                          {moment(info.row.original.lastUsed).from()}
                        </span>
                      </div>
                    )}
                    <div className='space-y-1 pt-2 text-3xs text-gray-500 sm:hidden'>
                      the key will expire on{' '}
                      <span className='font-semibold text-gray-700'>
                        {moment(info.row.original.expires).format('YYYY/MM/DD')}
                      </span>
                    </div>
                  </div>
                )
              },
              header: () => <span>Name</span>,
              accessorKey: 'name',
            },
            {
              cell: info => (
                <div className='hidden sm:table-cell'>
                  {info.getValue() ? moment(info.getValue()).from() : '-'}
                </div>
              ),
              header: () => (
                <span className='hidden sm:table-cell'>Created</span>
              ),
              accessorKey: 'created',
            },
            {
              cell: info => (
                <div className='hidden sm:table-cell'>
                  {info.getValue() ? moment(info.getValue()).from() : '-'}
                </div>
              ),
              header: () => (
                <span className='hidden sm:table-cell'>Last used</span>
              ),
              accessorKey: 'lastUsed',
            },
            {
              cell: info => (
                <div className='hidden sm:table-cell'>
                  {info.getValue() ? moment(info.getValue()).from() : '-'}
                </div>
              ),
              header: () => (
                <span className='hidden sm:table-cell'>Expires</span>
              ),
              accessorKey: 'expires',
            },
            {
              id: 'delete',
              cell: function Cell(info) {
                const [openDeleteModal, setOpenDeleteModal] = useState(false)

                const { name, id } = info.row.original

                return (
                  <div className='flex justify-end'>
                    <button
                      type='button'
                      onClick={() => {
                        setOpenDeleteModal(true)
                      }}
                      className='group flex w-full items-center rounded-md bg-white px-2 py-1.5 text-xs font-medium text-red-500'
                    >
                      <TrashIcon className='mr-2 h-3.5 w-3.5' />
                      <span className='hidden sm:block'>Remove</span>
                    </button>
                    <DeleteModal
                      open={openDeleteModal}
                      setOpen={setOpenDeleteModal}
                      primaryButtonText='Remove'
                      onSubmit={async () => {
                        await fetch(`/api/access-keys/${id}`, {
                          method: 'DELETE',
                        })
                        setOpenDeleteModal(false)
                        mutate()
                      }}
                      title='Remove Access Key'
                      message={
                        <div>
                          Are you sure you want to remove access key:{' '}
                          <span className='break-all font-bold'>{name}</span>?
                        </div>
                      }
                    />
                  </div>
                )
              },
            },
          ]}
        />
      </div>
    </>
  )
}

function AccessKeys() {
  const { isAdmin } = useUser()

  return (
    <>
      <div className='space-y-16'>
        {isAdmin && (
          <div>
            <ConnectorKeys />
          </div>
        )}
        <div>
          <PersonalKeys />
        </div>
      </div>
    </>
  )
}

function Password() {
  const { user } = useUser()
  const [oldPassword, setOldPassword] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState('')
  const [errors, setErrors] = useState({})
  const [showNotification, setShowNotification] = useState(false)
  const timerRef = useRef(null)

  useEffect(() => {
    return clearTimer()
  }, [])

  function clearTimer() {
    setShowNotification(false)
    return clearTimeout(timerRef.current)
  }

  async function onSubmit(e) {
    const submitButton = e.currentTarget

    e.preventDefault()
    submitButton.disabled = true

    if (password !== confirmPassword) {
      setErrors({
        confirmPassword: 'passwords do not match',
      })
      return false
    }

    setError('')
    setErrors({})

    try {
      const rest = await fetch(`/api/users/${user?.id}`, {
        method: 'PUT',
        body: JSON.stringify({
          ...user,
          oldPassword,
          password: confirmPassword,
        }),
      })

      const data = await rest.json()

      if (!rest.ok) {
        throw data
      }

      setOldPassword('')
      setPassword('')
      setConfirmPassword('')

      setShowNotification(true)
      setTimeout(() => {
        setShowNotification(false)
      }, 5000)
    } catch (e) {
      submitButton.disabled = false

      if (e.fieldErrors) {
        const errors = {}
        for (const error of e.fieldErrors) {
          const fieldName = error.fieldName.toLowerCase()
          if (fieldName === 'password') {
            errors[fieldName] = formatPasswordRequirements(error.errors)
          } else {
            errors[fieldName] = error.errors[0] || 'invalid value'
          }
        }
        setErrors(errors)
      } else {
        setError(e.message)
      }
    }
  }

  return (
    <form onSubmit={onSubmit} className='flex flex-col'>
      <div className='relative w-full space-y-3'>
        <div>
          <label
            htmlFor='old-password'
            className='text-2xs font-medium text-gray-700'
          >
            Old Password
          </label>
          <input
            required
            name={'old-password'}
            type='password'
            autoComplete='off'
            value={oldPassword}
            onChange={e => {
              setOldPassword(e.target.value)
              setErrors({})
              setError('')
            }}
            className={`mt-1 block w-full rounded-md shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm ${
              errors.oldpassword ? 'border-red-500' : 'border-gray-300'
            }`}
          />
          {errors.oldpassword && (
            <p className='my-1 text-xs text-red-500'>{errors.oldpassword}</p>
          )}
        </div>
        <div>
          <label
            htmlFor='password'
            className='text-2xs font-medium text-gray-700'
          >
            New Password
          </label>
          <input
            required
            name={'password'}
            type='password'
            autoComplete='off'
            value={password}
            onChange={e => {
              setPassword(e.target.value)
              setErrors({})
              setError('')
            }}
            className={`mt-1 block w-full rounded-md shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm ${
              errors.password ? 'border-red-500' : 'border-gray-300'
            }`}
          />
          {errors.password && (
            <p className='my-1 text-xs text-red-500'>{errors.password}</p>
          )}
        </div>
        <div>
          <label htmlFor={name} className='text-2xs font-medium text-gray-700'>
            Confirm New Password
          </label>
          <input
            required
            name={'password'}
            type='password'
            autoComplete='off'
            value={confirmPassword}
            onChange={e => {
              setConfirmPassword(e.target.value)
              setErrors({})
              setError('')
            }}
            className={`mt-1 block w-full rounded-md shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm ${
              errors.confirmPassword ? 'border-red-500' : 'border-gray-300'
            }`}
          />
          {errors.confirmPassword && (
            <p className='my-1 text-xs text-red-500'>
              {errors.confirmPassword}
            </p>
          )}
        </div>
      </div>
      <div className='mt-6 flex flex-row items-center justify-end space-x-3'>
        <button
          type='submit'
          disabled={
            !(
              oldPassword &&
              password &&
              confirmPassword &&
              Object.keys(errors).length === 0 &&
              error === ''
            )
          }
          className='inline-flex cursor-pointer items-center rounded-md border border-transparent bg-black px-4 py-2 text-xs font-medium text-white shadow-sm hover:bg-gray-800 focus:outline-none focus:ring-2 focus:ring-black focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-30'
        >
          Reset Password
        </button>
      </div>
      {error && <p className='my-1 text-xs text-red-500'>{error}</p>}
      <Notification
        show={showNotification}
        setShow={setShowNotification}
        text='Password Successfully Reset'
        setClearNotification={() => clearTimer()}
      />
    </form>
  )
}

function Admins() {
  const { user } = useUser()
  const { data: { items: users } = {} } = useSWR('/api/users?limit=1000')
  const { data: { items: groups } = {} } = useSWR('/api/groups?limit=1000')
  const { data: { items: grants } = {}, mutate } = useSWR(
    '/api/grants?resource=infra&privilege=admin&limit=1000'
  )
  const { data: { items: selfGroups } = {} } = useSWR(
    () => `/api/groups?userID=${user?.id}&limit=1000`
  )

  const grantsList = grants?.sort(sortBySubject)?.map(grant => {
    const message =
      grant?.user === user?.id
        ? 'Are you sure you want to remove yourself as an admin?'
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
    <>
      <p className='mt-1 mb-4 text-xs text-gray-500'>
        These users and groups have full access to this organization.
      </p>
      <div className='mb-5 w-full rounded-lg border border-gray-200/75 px-5 py-3'>
        <GrantForm
          resource='infra'
          roles={['admin']}
          grants={grants}
          multiselect={false}
          onSubmit={async ({ user, group }) => {
            if (grants?.find(g => g.user === user && g.group === group)) {
              return false
            }

            await fetch('/api/grants', {
              method: 'POST',
              body: JSON.stringify({
                user,
                group,
                privilege: 'admin',
                resource: 'infra',
              }),
            })
            mutate()
          }}
        />
      </div>
      <Table
        data={grantsList}
        columns={[
          {
            cell: function Cell(info) {
              return (
                <div className='flex flex-col'>
                  <div className='flex items-center font-medium text-gray-700'>
                    {info.getValue()}
                  </div>
                  <div className='text-2xs text-gray-500'>
                    {info.row.original.user && 'User'}
                    {info.row.original.group && 'Group'}
                  </div>
                </div>
              )
            },
            header: () => <span>Admin</span>,
            accessorKey: 'name',
          },
          {
            cell: function Cell(info) {
              const [open, setOpen] = useState(false)
              const [deleteId, setDeleteId] = useState(null)

              return (
                grants?.length > 1 && (
                  <div className='text-right'>
                    <button
                      onClick={() => {
                        setDeleteId(info.row.original.id)
                        setOpen(true)
                      }}
                      className='p-1 text-2xs text-gray-500/75 hover:text-gray-600'
                    >
                      Revoke
                      <span className='sr-only'>{info.row.original.name}</span>
                    </button>
                    <DeleteModal
                      open={open}
                      setOpen={setOpen}
                      primaryButtonText='Revoke'
                      onSubmit={async () => {
                        await fetch(`/api/grants/${deleteId}`, {
                          method: 'DELETE',
                        })
                        setOpen(false)
                      }}
                      title='Revoke Admin'
                      message={
                        !grantsList?.find(grant => grant.id === deleteId)
                          ?.message ? (
                          <>
                            Are you sure you want to revoke admin access for{' '}
                            <span className='font-bold'>
                              {
                                grantsList?.find(grant => grant.id === deleteId)
                                  ?.name
                              }
                            </span>
                            ?
                          </>
                        ) : (
                          grantsList?.find(grant => grant.id === deleteId)
                            ?.message
                        )
                      }
                    />
                  </div>
                )
              )
            },
            id: 'delete',
          },
        ]}
      />
    </>
  )
}

function Authentication() {
  const { data: { items: providers } = {} } = useSWR(
    `/api/providers?&limit=1000`
  )
  const { data: org } = useSWR('/api/organizations/self')
  const [allowedDomains, setAllowedDomains] = useState(
    org?.allowedDomains || []
  )
  const [newDomain, setNewDomain] = useState('')
  const [error, setError] = useState('')

  async function removeDomain(e) {
    e.preventDefault()
    let toRemove = e.target.value
    const newAllowedDomains = allowedDomains.filter(d => d !== toRemove)
    updateAllowedDomains(newAllowedDomains)
  }

  async function addDomain(e) {
    e.preventDefault()
    // check that we only have the domain (no protocol), but be lenient on validation
    let cleanedInput = newDomain
    if (newDomain.startsWith('http://')) {
      cleanedInput = cleanedInput.replace('http://', '')
    }
    if (newDomain.startsWith('https://')) {
      cleanedInput = cleanedInput.replace('https://', '')
    }
    if (newDomain.startsWith('www.')) {
      cleanedInput = cleanedInput.replace('www.', '')
    }
    if (newDomain.startsWith('@')) {
      cleanedInput = cleanedInput.replace('@', '')
    }
    if (!allowedDomains.includes(cleanedInput)) {
      const newAllowedDomains = [...allowedDomains, cleanedInput]
      updateAllowedDomains(newAllowedDomains)
    }
    setNewDomain('')
  }

  function onKeyDown(e) {
    const { key } = e

    if (key === 'Backspace' && newDomain === '' && allowedDomains.length > 0) {
      e.preventDefault()
      const newAllowedDomains = [...allowedDomains]
      newAllowedDomains.pop()
      updateAllowedDomains(newAllowedDomains)
    }
  }

  async function updateAllowedDomains(allowedDomains) {
    setError('')
    try {
      const res = await fetch('/api/organizations/' + org.id, {
        method: 'PUT',
        body: JSON.stringify({ allowedDomains }),
      })
      await jsonBody(res)
      setAllowedDomains(allowedDomains)
      mutate('/api/organizations/self')
    } catch (e) {
      setError(e.message)
    }
  }

  return (
    <>
      {providers?.some(p => p.id === googleSocialLoginID) && (
        <>
          <header className='my-6 flex flex-col justify-between md:flex-row md:space-y-0 md:space-x-4'>
            <div>
              <h2 className='mb-0.5 flex items-center font-display text-lg font-medium'>
                Allowed Domains
              </h2>
              <h3 className='text-sm text-gray-500'>
                Google accounts with these domains are able to log in to your
                organization. They will not have any infrastructure access by
                default.
              </h3>
            </div>
          </header>
          <form
            className='group form-input flex w-full rounded border-gray-300 py-2 px-3 leading-tight focus-within:border-blue-500 focus-within:ring-blue-500'
            onSubmit={addDomain}
          >
            {allowedDomains.map(d => (
              <span
                className='mr-1 inline-block rounded-md bg-gray-200 py-1 px-2 pl-2.5 pr-1 text-xs font-medium'
                key={d}
              >
                {d}
                <button
                  className='px-1.5 font-normal hover:text-gray-600'
                  type='button'
                  value={d}
                  onClick={removeDomain}
                >
                  ✕
                </button>
              </span>
            ))}
            <input
              className='peer bg-transparent focus:outline-none'
              value={newDomain}
              onChange={e => {
                setNewDomain(e.target.value)
              }}
              onKeyDown={onKeyDown}
            />
          </form>
          {error && <p className='my-1 text-xs text-red-500'>{error}</p>}
        </>
      )}

      <header className='my-6 flex flex-col justify-between space-y-4 md:flex-row md:space-y-0 md:space-x-4'>
        <div>
          <h2 className='mb-0.5 flex items-center font-display text-lg font-medium'>
            Identity Providers
          </h2>
          <h3 className='text-sm text-gray-500'>
            Configure additional methods of logging in using custom OpenID
            Connect (OIDC) identity providers.
          </h3>
        </div>
        <Link
          href='/settings/providers/add'
          className='inline-flex items-center self-end whitespace-nowrap rounded-md border border-transparent bg-black px-3 py-2 text-xs font-medium text-white shadow-sm hover:cursor-pointer hover:bg-gray-800'
        >
          <PlusIcon className='mr-1 h-3 w-3' /> Connect provider
        </Link>
      </header>
      <div className='mt-3 flex min-h-0 flex-1 flex-col'>
        <Table
          href={row => `/settings/providers/${row.original.id}`}
          data={providers?.filter(p => p.id !== googleSocialLoginID)} // remove google login provider
          empty='No providers'
          columns={[
            {
              cell: info => (
                <div className='flex flex-row items-center py-1'>
                  <div className='mr-3 flex h-9 w-9 flex-none items-center justify-center rounded-md border border-gray-200'>
                    <img
                      alt='provider icon'
                      className='h-4'
                      src={`/providers/${info.row.original.kind}.svg`}
                    />
                  </div>
                  <div className='flex flex-col'>
                    <div className='text-sm font-medium text-gray-700'>
                      {info.getValue()}
                    </div>
                    <div className='text-2xs text-gray-500 sm:hidden'>
                      {info.row.original.url}
                    </div>
                    <div className='font-mono text-2xs text-gray-400 lg:hidden'>
                      {info.row.original.clientID}
                    </div>
                  </div>
                </div>
              ),
              header: () => <span>Name</span>,
              accessorKey: 'name',
            },
            {
              cell: info => (
                <div className='hidden sm:table-cell'>{info.getValue()}</div>
              ),
              header: () => <span className='hidden sm:table-cell'>URL</span>,
              accessorKey: 'url',
            },
            {
              cell: info => (
                <div className='hidden font-mono lg:table-cell'>
                  {info.getValue()}
                </div>
              ),
              header: () => (
                <span className='hidden lg:table-cell'>Client ID</span>
              ),
              accessorKey: 'clientID',
            },
          ]}
        />
      </div>
    </>
  )
}

export default function Settings() {
  const router = useRouter()
  const { user, isAdmin } = useUser()
  const hasInfraProvider = user?.providerNames?.includes('infra')

  const tabs = [
    {
      name: 'access_keys',
      title: 'Access Keys',
      render: <AccessKeys />,
    },
    ...(hasInfraProvider
      ? [
          {
            name: 'password',
            title: 'Reset Password',
            render: <Password />,
          },
        ]
      : []),
    ...(isAdmin
      ? [
          {
            name: 'admins',
            title: 'Admins',
            render: <Admins />,
          },
          {
            name: 'authentication',
            title: 'Authentication',
            render: <Authentication />,
          },
        ]
      : []),
  ]

  const tab = router.query.tab || tabs[0].name

  return (
    <div className='my-6'>
      <Head>
        <title>Settings - Infra</title>
      </Head>
      <div className='flex flex-1 flex-col'>
        {/* Header */}
        <h1 className='mb-6 font-display text-xl font-medium'>Settings</h1>

        {/* Tabs */}
        {tabs.length > 0 && (
          <div>
            <div className='mb-4 md:hidden'>
              <label htmlFor='tabs' className='sr-only'>
                Select a tab from settings page
              </label>
              <Menu as='div' className='relative inline-block w-full text-left'>
                <Menu.Button className='inline-flex w-full items-center justify-between rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 shadow-sm hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 focus:ring-offset-gray-100'>
                  {tabs.find(t => t.name === tab).title}
                  <ChevronDownIcon
                    className='ml-2 h-4 w-4'
                    aria-hidden='true'
                  />
                </Menu.Button>

                <Menu.Items className='absolute right-0 z-10 mt-2 w-full origin-top-right rounded-md bg-white shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none'>
                  <div className='py-1'>
                    {tabs.map(t => (
                      <Menu.Item key={t.name}>
                        {({ active }) => (
                          <Link
                            href={{
                              pathname: '/settings',
                              query: { tab: t.name },
                            }}
                            className={`
                            ${
                              active
                                ? 'bg-gray-100 text-gray-900'
                                : 'text-gray-700'
                            }
                            flex items-center justify-between px-4 py-2 text-sm`}
                          >
                            {t.title}
                            {tab === t.name && (
                              <CheckIcon
                                className='h-3 w-3 text-gray-900'
                                aria-hidden='true'
                              />
                            )}
                          </Link>
                        )}
                      </Menu.Item>
                    ))}
                  </div>
                </Menu.Items>
              </Menu>
            </div>
            <div className='hidden md:block'>
              <div className='mb-3 border-b border-gray-200'>
                <nav className='-mb-px flex' aria-label='Tabs'>
                  {tabs.map(t => (
                    <Link
                      key={t.name}
                      href={{
                        pathname: `/settings/`,
                        query: { tab: t.name },
                      }}
                      className={`
                ${
                  tab === t.name
                    ? 'border-blue-500 text-blue-600'
                    : 'border-transparent text-gray-500 hover:text-gray-600'
                }
                 whitespace-nowrap border-b-2 py-2 text-sm font-medium capitalize md:px-6 lg:px-8`}
                      aria-current={tab.current ? 'page' : undefined}
                    >
                      {t.title}
                    </Link>
                  ))}
                </nav>
              </div>
            </div>
            <div className='my-10'>
              {tabs.map(
                t =>
                  tab === t.name &&
                  t.render && (
                    <React.Fragment key={t.name}>{t.render}</React.Fragment>
                  )
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
Settings.layout = page => {
  return <Dashboard>{page}</Dashboard>
}
