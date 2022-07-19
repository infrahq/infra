import { useRouter } from 'next/router'

import { ChevronLeftIcon, ChevronRightIcon } from '@heroicons/react/solid'

function CurrentPage(path, page) {
  return (
    <a
      href={path + '?p=' + page}
      className='inline-flex items-center rounded-md bg-gray-700 py-2 px-4 text-sm text-white hover:text-violet-300'
      aria-current='page'
      key={page}
    >
      {page}
    </a>
  )
}

function Page(path, page) {
  return (
    <a
      href={path + '?p=' + page}
      className='inline-flex items-center py-2 px-4 text-sm font-medium text-gray-500 hover:text-violet-300'
      key={page}
    >
      {page}
    </a>
  )
}

function LeftArrow(path, page) {
  return (
    <a
      href={path + '?p=' + (page > 1 ? page - 1 : 1)}
      className=' inline-flex items-center pr-1 text-sm font-medium text-gray-500 hover:text-violet-300 '
    >
      <ChevronLeftIcon
        className=' h-5 w-5 text-gray-400 hover:text-violet-300'
        aria-hidden='true'
      />
    </a>
  )
}

function RightArrow(path, page, maxPage) {
  console.log(maxPage)
  return (
    <a
      href={path + '?p=' + (page < maxPage ? page + 1 : Math.max(1, maxPage))}
      className='inline-flex items-center pl-1 text-sm font-medium text-gray-500 hover:text-violet-300 '
    >
      <ChevronRightIcon
        className='ml-3 h-5 w-5 text-gray-400 hover:text-violet-300'
        aria-hidden='true'
      />
    </a>
  )
}

export default function Pagination({ curr, totalPages, totalCount }) {
  const router = useRouter()
  const path = router.pathname
  const limit = 13

  curr = curr === undefined ? 1 : parseInt(curr)
  totalPages = totalPages === undefined ? 1 : parseInt(totalPages)
  totalCount = totalCount === undefined ? 0 : parseInt(totalCount)

  let pages = [LeftArrow(path, curr)]

  for (
    let page = Math.max(1, curr - 3);
    pages.length < 7 && page <= totalPages;
    page++
  ) {
    if (page === curr) {
      pages.push(CurrentPage(path, page))
    } else {
      pages.push(Page(path, page))
    }
  }
  pages.push(RightArrow(path, curr, totalPages))

  let lowerItem = totalCount === 0 ? 0 : 1+(curr-1)*limit
  let upperItem = Math.min(1+(curr-1)*limit + limit, totalCount)

  return (  
    [
    <nav key='paginator'className='flex justify-end px-4'>{pages}</nav>,<h3 key='results' className='flex justify-end pb-4 px-4 text-3xs text-gray-400'> Displaying {lowerItem}–{upperItem} out of {totalCount}</h3>])
}
