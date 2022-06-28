import { useContext } from 'react'
import { TabContext } from './Tabs'

export default function Tab({ label, children }) {
  const currentTab = useContext(TabContext)

  if (label !== currentTab) {
    return null
  }

  return children
}
