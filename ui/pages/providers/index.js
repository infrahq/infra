import useSWR from 'swr'
import Head from 'next/head'
import { useRouter } from 'next/router'
import { ViewGridIcon } from '@heroicons/react/outline'
import { useState } from 'react'

import { useAdmin } from '../../lib/admin'

import Dashboard from '../../components/layouts/dashboard'
import EmptyTable from '../../components/empty-table'
import Pagination from '../../components/pagination'
import PageHeader from '../../components/page-header'
import DeleteModal from '../../components/delete-modal'

function ProviderTable({ providers, mutate }) {
  const [modalOpen, setModalOpen] = useState(false)
  const [selectedProvider, setSelectedProvider] = useState(null)

  return (
    <div className='overflow-hidden shadow ring-1 ring-black ring-opacity-5 md:rounded-lg'>
      <table className='min-w-full divide-y divide-gray-300'>
        <thead className='bg-gray-50'>
          <tr>
            <th
              scope='col'
              className='py-3.5 pl-4 pr-3 text-left text-sm font-semibold text-gray-900 sm:pl-6'
            >
              Name
            </th>
            <th
              scope='col'
              className='hidden px-3 py-3.5 text-left text-sm font-semibold text-gray-900 lg:table-cell'
            >
              URL
            </th>
            <th
              scope='col'
              className='hidden px-3 py-3.5 text-left text-sm font-semibold text-gray-900 sm:table-cell'
            >
              Client Id
            </th>
            <th scope='col' className='relative py-3.5 pl-3 pr-4 sm:pr-6'>
              <span className='sr-only'>Remove</span>
            </th>
          </tr>
        </thead>
        <tbody className='divide-y divide-gray-200 bg-white'>
          {providers?.map(provider => (
            <tr key={provider.id}>
              <td className='w-full max-w-0 py-4 pl-4 pr-3 text-sm font-medium text-gray-900 sm:w-auto sm:max-w-none sm:pl-6'>
                <div className='flex items-center py-1.5'>
                  <div className='flex h-7 w-7 flex-none items-center justify-center rounded-md border border-gray-200'>
                    <img
                      alt='provider icon'
                      className='h-3'
                      src={`/providers/${provider.kind}.svg`}
                    />
                  </div>
                  <div className='ml-3 truncate py-1 text-2xs sm:max-w-[10rem]'>
                    {provider.name}
                  </div>
                </div>
                <dl className='font-normal lg:hidden'>
                  <dt className='sr-only'>URL</dt>
                  <dd className='mt-1 truncate text-gray-700'>
                    {provider.url}
                  </dd>
                  <dt className='sr-only sm:hidden'>Client Id</dt>
                  <dd className='mt-1 truncate text-gray-500 sm:hidden'>
                    {provider.clientID}
                  </dd>
                </dl>
              </td>
              <td className='hidden px-3 py-4 text-sm text-gray-500 sm:max-w-[10rem] lg:table-cell'>
                {provider.url}
              </td>
              <td className='hidden truncate px-3 py-4 text-sm text-gray-500 sm:table-cell sm:max-w-[10rem]'>
                {provider.clientID}
              </td>
              <td className='py-4 pl-3 pr-4 text-right text-sm font-medium sm:pr-6'>
                <button
                  onClick={() => {
                    setModalOpen(true)
                    setSelectedProvider(provider)
                  }}
                  className='text-xs text-blue-600 hover:text-blue-900'
                >
                  Remove<span className='sr-only'>, {provider.name}</span>
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      <DeleteModal
        open={modalOpen}
        setOpen={setModalOpen}
        onSubmit={async () => {
          await fetch(`/api/providers/${selectedProvider.id}`, {
            method: 'DELETE',
          })
          setModalOpen(false)

          mutate({ items: providers.filter(p => p.id !== selectedProvider.id) })
        }}
        title='Remove Identity Provider'
        message={
          <>
            Are you sure you want to delete{' '}
            <span className='font-bold'>{selectedProvider?.name}</span>? This
            action cannot be undone.
          </>
        }
      />
    </div>
  )
}

export default function Providers() {
  const router = useRouter()
  const page = router.query.p === undefined ? 1 : router.query.p
  const limit = 13
  const {
    data: { items: providers, totalPages, totalCount } = {},
    mutate,
    error,
  } = useSWR(`/api/providers?page=${page}&limit=${limit}`)
  const { admin, loading: adminLoading } = useAdmin()

  const loading = adminLoading || (!providers && !error)

  return (
    <div className='md:px-6 xl:px-10 2xl:m-auto 2xl:max-w-6xl'>
      <Head>
        <title>Identity Providers - Infra</title>
      </Head>
      <div className='pb-6'>
        <PageHeader
          buttonHref={admin && '/providers/add'}
          buttonLabel='Provider'
        />
      </div>

      <div className='px-4 sm:px-6 md:px-0'>
        {!loading && (
          <div className='flex flex-1 flex-col space-y-4'>
            {error?.status ? (
              <div className='my-20 text-center text-sm font-light text-gray-300'>
                {error?.info?.message}
              </div>
            ) : (
              <div className='flex min-h-0 flex-1 flex-col px-0 md:px-6 xl:px-0'>
                <ProviderTable providers={providers} mutate={mutate} />
                {providers?.length === 0 && (
                  <EmptyTable
                    title='There are no providers'
                    subtitle={
                      <>
                        Identity providers allow you to connect your existing
                        users &amp; groups to Infra.
                      </>
                    }
                    iconPath='/providers.svg'
                    icon={<ViewGridIcon />}
                  />
                )}
              </div>
            )}
            {totalPages > 1 && (
              <Pagination
                curr={page}
                totalPages={totalPages}
                totalCount={totalCount}
                limit={limit}
              ></Pagination>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

Providers.layout = function (page) {
  return <Dashboard>{page}</Dashboard>
}
