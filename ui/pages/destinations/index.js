import useSWR from 'swr'
import Head from 'next/head'
import { ChipIcon } from '@heroicons/react/outline'
import { useRouter } from 'next/router'

import { useAdmin } from '../../lib/admin'

import Breadcrumbs from '../../components/breadcrumbs'
import Dashboard from '../../components/layouts/dashboard'
import EmptyTable from '../../components/empty-table'
import Pagination from '../../components/pagination'
import PageHeader from '../../components/page-header'

function DestinationTable({ destinations }) {
  const router = useRouter()

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
              Namespaces
            </th>
            <th
              scope='col'
              className='hidden px-3 py-3.5 text-left text-sm font-semibold text-gray-900 lg:table-cell'
            >
              Version
            </th>
            <th
              scope='col'
              className='hidden px-3 py-3.5 text-left text-sm font-semibold text-gray-900 sm:table-cell'
            >
              Status
            </th>
          </tr>
        </thead>
        <tbody className='divide-y divide-gray-200 bg-white'>
          {destinations?.map(destination => (
            <tr
              key={destination.id}
              onClick={() => router.replace(`/destinations/${destination.id}`)}
              className='hover:cursor-pointer hover:bg-gray-100'
            >
              <td className='w-full max-w-0 py-4 pl-4 pr-3 text-sm font-medium text-gray-900 sm:w-auto sm:max-w-none sm:pl-6'>
                <div className='flex items-center py-1.5'>
                  <div className='text-sm sm:max-w-[10rem]'>
                    {destination.name}
                  </div>
                </div>
                <dl className='font-normal lg:hidden'>
                  <dt className='sr-only'>Namespaces</dt>
                  <dd className='mt-1 max-w-[10rem] text-gray-700'>
                    {destination.resources ? (
                      <>
                        {destination.resources.length}{' '}
                        {destination.resources.length === 1
                          ? 'namespace'
                          : 'namespaces'}
                      </>
                    ) : (
                      '-'
                    )}
                  </dd>
                  <dt className='sr-only'>Version</dt>
                  <dd className='mt-1 max-w-[10rem] text-gray-700'>
                    {destination?.version ? <>{destination.version}</> : '-'}
                  </dd>
                  <dt className='sr-only sm:hidden'>Status</dt>
                  <dd className='mt-1 text-gray-500 sm:hidden'>
                    <div className='flex items-center py-2'>
                      <div
                        className={`h-2 w-2 flex-none rounded-full ${
                          destination.connected ? 'bg-green-500' : 'bg-gray-600'
                        }`}
                      />
                      <span className='flex-none px-2 text-gray-700'>
                        {destination.connected ? 'Connected' : 'Disconnected'}
                      </span>
                    </div>
                  </dd>
                </dl>
              </td>
              <td className='hidden truncate px-3 py-4 text-sm text-gray-700 sm:max-w-[10rem] lg:table-cell'>
                {destination.resources ? destination.resources.length : '-'}
              </td>
              <td className='hidden truncate px-3 py-4 text-sm text-gray-700 sm:max-w-[10rem] lg:table-cell'>
                {destination?.version ? <>{destination.version}</> : '-'}
              </td>
              <td className='hidden truncate px-3 py-4 text-sm text-gray-700 sm:table-cell sm:max-w-[10rem]'>
                <div className='flex items-center py-2'>
                  <div
                    className={`h-2 w-2 flex-none rounded-full ${
                      destination.connected ? 'bg-green-500' : 'bg-gray-600'
                    }`}
                  />
                  <span className='flex-none px-2 text-gray-700'>
                    {destination.connected ? 'Connected' : 'Disconnected'}
                  </span>
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

export default function Destinations() {
  const router = useRouter()
  const page = router.query.p === undefined ? 1 : router.query.p
  const limit = 13 // TODO: make limit dynamic

  const { data: { items: destinations, totalPages, totalCount } = {}, error } =
    useSWR(`/api/destinations?page=${page}&limit=${limit}`)
  const { admin, loading: adminLoading } = useAdmin()

  const data = destinations?.map(d => ({
    ...d,
    kind: 'cluster',
    resource: d.name,

    // Create "fake" destinations as subrows from resources
    subRows: d.resources?.map(r => ({
      name: r,
      resource: `${d.name}.${r}`,
      kind: 'namespace',
      roles: d.roles?.filter(r => r !== 'cluster-admin'),
    })),
  }))

  const loading = adminLoading || !data

  return (
    <div className='md:px-6 xl:px-10 2xl:m-auto 2xl:max-w-6xl'>
      <Head>
        <title>Clusters - Infra</title>
      </Head>
      <Breadcrumbs>{'Clusters'}</Breadcrumbs>
      <div className='py-6'>
        <PageHeader
          buttonHref={admin && '/destinations/add'}
          buttonLabel='Cluster'
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
                <DestinationTable destinations={destinations} />
                {destinations?.length === 0 && (
                  <EmptyTable
                    title='There are no clusters'
                    subtitle='There is currently no cluster connected to Infra'
                    iconPath='/destinations.svg'
                    icon={<ChipIcon />}
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

Destinations.layout = function (page) {
  return <Dashboard>{page}</Dashboard>
}
