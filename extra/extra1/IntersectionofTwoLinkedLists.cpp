class Solution {
public:
    // リストの長さを計算する
    int getLength(ListNode *head){
        int length = 0;
        ListNode *current = head;
        while(current != NULL){
            length ++;
            current = current -> next;
        }
        return length;
    }
    ListNode *getIntersectionNode(ListNode *headA, ListNode *headB) {
        int lengthA = getLength(headA);
        int lengthB = getLength(headB);
        int diff = abs(lengthA - lengthB);
    
        if(lengthA > lengthB){
            for(int i=0;i<diff;i++) headA = headA -> next;
        }else{
            for(int i=0;i<diff;i++) headB = headB -> next;
        }
        
        while(headA != NULL && headB != NULL){
            if(headA == headB){
                return headA;
            }
            headA = headA -> next;
            headB = headB -> next;
        }
        return NULL;
    }
};
