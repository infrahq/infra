import {
  flexRender,
  getCoreRowModel,
  useReactTable,
} from '@tanstack/react-table'
import Link from 'next/link'

export default function Table({
  columns,
  data,
  href,
  empty = 'No data',
  count = data?.length,

  // TODO: default to something better – i.e. automatic pagination
  pageSize = 999,
  pageIndex = 0,
  pageCount = 1,
  onPageChange,
}) {
  const table = useReactTable({
    data,
    columns,
    getCoreRowModel: getCoreRowModel(),
    pageCount,
    state: {
      pagination: {
        pageSize,
        pageIndex,
      },
    },
    onPaginationChange: f =>
      onPageChange(
        f({
          pageSize,
          pageIndex,
        })
      ),
    manualPagination: true,
  })

  return (
    <div className='overflow-visible rounded-lg border border-gray-200/75'>
      <table className='w-full text-sm text-gray-600'>
        <thead className='border-b border-gray-200/75 bg-zinc-50/50 text-xs text-gray-500'>
          {table.getHeaderGroups().map(headerGroup => (
            <tr key={headerGroup.id}>
              {headerGroup.headers.map(header => (
                <th
                  className='py-2 px-5 text-left font-medium first:w-[35%]'
                  key={header.id}
                >
                  {header.isPlaceholder
                    ? null
                    : flexRender(
                        header.column.columnDef.header,
                        header.getContext()
                      )}
                </th>
              ))}
            </tr>
          ))}
        </thead>
        <tbody className='divide-y divide-gray-100'>
          {data &&
            table.getRowModel().rows.map(row => (
              <tr
                className={`group ${
                  href ? 'cursor-pointer hover:bg-gray-50/50' : ''
                }`}
                key={row.id}
              >
                {row.getVisibleCells().map(cell => (
                  <td
                    className={`border-gray-100 text-sm sm:max-w-[10rem] ${
                      href ? '' : 'px-5 py-2.5'
                    }`}
                    key={cell.id}
                  >
                    {href ? (
                      <Link href={href(row)}>
                        <a tabIndex='-1' className='block px-5 py-2.5'>
                          {flexRender(
                            cell.column.columnDef.cell,
                            cell.getContext()
                          )}
                        </a>
                      </Link>
                    ) : (
                      flexRender(cell.column.columnDef.cell, cell.getContext())
                    )}
                  </td>
                ))}
              </tr>
            ))}
        </tbody>
      </table>

      {/* Pagination */}
      {data?.length > 0 && (
        <div className='sticky left-0 z-0 flex w-full items-center justify-between border-t border-gray-200/75 py-2 px-5 text-2xs'>
          <div className='text-gray-500'>
            Showing{' '}
            {count === 1
              ? '1'
              : `${pageIndex * pageSize + 1} – ${Math.min(
                  (pageIndex + 1) * pageSize,
                  count
                )}`}{' '}
            of {count} result
            {count === 1 ? '' : 's'}
          </div>
          <div className='space-x-1 text-2xs text-gray-700'>
            <button
              className='rounded-md border border-gray-200 px-3 py-1 hover:bg-gray-50 disabled:cursor-default disabled:border-gray-100 disabled:text-gray-300 disabled:hover:bg-white'
              onClick={() => table.previousPage()}
              disabled={!table.getCanPreviousPage()}
            >
              Prev
            </button>
            <button
              className='rounded-md border border-gray-200 px-3 py-1 hover:bg-gray-50 disabled:cursor-default disabled:border-gray-100 disabled:text-gray-300 disabled:hover:bg-white'
              onClick={() => table.nextPage()}
              disabled={!table.getCanNextPage()}
            >
              Next
            </button>
          </div>
        </div>
      )}
      {data && data.length === 0 && empty && (
        <div className='flex justify-center py-5 text-sm text-gray-500'>
          {empty}
        </div>
      )}
      {!data && (
        <div className='flex min-h-[100px] items-center justify-center py-4 text-xs text-gray-400'>
          <svg
            xmlns='http://www.w3.org/2000/svg'
            viewBox='0 0 100 100'
            preserveAspectRatio='xMidYMid'
            className='h-10 w-10 animate-spin-fast stroke-current text-gray-400'
          >
            <circle
              cx='50'
              cy='50'
              fill='none'
              strokeWidth='1.5'
              r='24'
              strokeDasharray='113.09733552923255 39.69911184307752'
            ></circle>
          </svg>
        </div>
      )}
    </div>
  )
}
