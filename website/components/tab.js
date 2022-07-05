import { useContext } from 'react'
import { TabContext } from './tabs'

export default function Tab({ label, children }) {
  const currentTab = useContext(TabContext)

  if (label !== currentTab) {
    return null
  }

  return children
}
