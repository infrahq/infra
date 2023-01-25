import Link from 'next/link'
import { useLayoutEffect, useRef, useState } from 'react'

import { TrashIcon } from '@heroicons/react/24/outline'
import {
  flexRender,
  getCoreRowModel,
  useReactTable,
} from '@tanstack/react-table'

import Loader from './loader'

export default function Table({
  columns,
  data,
  href,
  type = '',
  empty = 'No data',
  count = data?.length,
  allowDelete = false,
  selectedRowIds = [],
  setSelectedRowIds = () => {},
  // TODO: default to something better – i.e. automatic pagination
  pageSize = 999,
  pageIndex = 0,
  pageCount = 1,
  onPageChange,
  onDelete,
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

  const checkbox = useRef()

  const [checkedAll, setCheckedAll] = useState(false)
  const [indeterminate, setIndeterminate] = useState(false)

  function toggleAll() {
    if (checkedAll || indeterminate) {
      setSelectedRowIds([])
    } else {
      setSelectedRowIds(data?.filter(d => !d.disabledCheckbox)?.map(d => d.id))
    }

    setCheckedAll(!checkedAll && !indeterminate)
    setIndeterminate(false)
  }

  useLayoutEffect(() => {
    const isIndeterminate =
      selectedRowIds.length > 0 && selectedRowIds.length < data?.length

    if (allowDelete && data?.length > 0) {
      setCheckedAll(selectedRowIds.length === data?.length)
      setIndeterminate(isIndeterminate)

      if (checkbox && checkbox.current) {
        checkbox.current.indeterminate = isIndeterminate
      }
    }
  }, [selectedRowIds])

  useLayoutEffect(() => {
    setCheckedAll(false)
    setIndeterminate(false)
    setSelectedRowIds([])
  }, [pageIndex])

  return (
    <div className='relative overflow-x-auto rounded-lg border border-gray-200/75'>
      {selectedRowIds.length > 0 && (
        <div className='absolute left-12 flex h-6 items-center py-4 sm:left-16'>
          <button
            type='button'
            onClick={() => onDelete()}
            className='rounded-md bg-zinc-50 px-4 py-2 text-2xs font-medium  text-red-500 hover:bg-red-100'
          >
            <div className='flex flex-row items-center'>
              <TrashIcon className='mr-1 mt-px h-3.5 w-3.5' />
              Remove selected
            </div>
          </button>
        </div>
      )}
      <table className='w-full text-sm text-gray-600'>
        <thead className='border-b border-gray-200/75 bg-zinc-50/50 text-xs text-gray-500'>
          {table.getHeaderGroups().map(headerGroup => {
            return (
              <tr key={headerGroup.id}>
                {allowDelete && data?.length > 0 && (
                  <th scope='col' className='relative w-5 px-6'>
                    <input
                      type='checkbox'
                      className='left-4 h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500 disabled:cursor-not-allowed disabled:opacity-30 disabled:focus:ring-0 sm:left-6'
                      ref={el => (checkbox.current = el)}
                      checked={checkedAll}
                      onChange={toggleAll}
                      disabled={
                        data?.length === 1 && data?.[0]?.disabledCheckbox
                      }
                    />
                  </th>
                )}
                {headerGroup.headers.map(header => (
                  <th
                    className='py-2 px-5 text-left font-medium'
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
            )
          })}
        </thead>
        <tbody className='divide-y divide-gray-100'>
          {data &&
            table.getRowModel().rows.map(row => {
              return (
                <tr
                  className={`group truncate ${
                    row.original.newCreate ? 'bg-yellow-100/50' : ''
                  } ${
                    href && href(row)
                      ? 'cursor-pointer hover:bg-gray-50/50'
                      : ''
                  } ${type === 'detail' ? 'h-[60px]' : ''}`}
                  key={row.id}
                >
                  {allowDelete && data?.length > 0 && (
                    <th scope='col'>
                      <input
                        type='checkbox'
                        className='left-4 h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500 disabled:cursor-not-allowed disabled:opacity-30 disabled:focus:ring-0 sm:left-6'
                        value={row.id}
                        checked={selectedRowIds.includes(row.original.id)}
                        disabled={row.original.disabledCheckbox}
                        onChange={e =>
                          setSelectedRowIds(
                            e.target.checked
                              ? [...selectedRowIds, row.original.id]
                              : selectedRowIds.filter(
                                  p => p !== row.original.id
                                )
                          )
                        }
                      />
                    </th>
                  )}
                  {row.getVisibleCells().map(cell => {
                    return (
                      <td
                        className={`border-gray-100 text-sm ${
                          href && href(row) ? '' : 'px-5 py-2'
                        }`}
                        key={cell.id}
                        {...{
                          style: {
                            minWidth: cell.column.getSize(),
                          },
                        }}
                      >
                        {href && href(row) ? (
                          <Link
                            href={href(row)}
                            tabIndex='-1'
                            className='block px-5 py-2'
                          >
                            {flexRender(
                              cell.column.columnDef.cell,
                              cell.getContext()
                            )}
                          </Link>
                        ) : (
                          flexRender(
                            cell.column.columnDef.cell,
                            cell.getContext()
                          )
                        )}
                      </td>
                    )
                  })}
                </tr>
              )
            })}
        </tbody>
      </table>

      {data && data.length === 0 && empty && (
        <div className='flex justify-center py-5 text-sm text-gray-500'>
          {empty}
        </div>
      )}
      {!data && (
        <div className='flex w-full justify-center'>
          <Loader className='h-12 w-12' />
        </div>
      )}
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
          <div className='flex items-center space-x-1 text-2xs text-gray-700'>
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
    </div>
  )
}
